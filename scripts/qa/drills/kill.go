package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AlexGalhardo/vukem-finances/scripts/qa/qakit"
)

type batch struct {
	worker int
	muts   []map[string]any
	uids   []string
	acked  bool
}

func (b *batch) payload() map[string]any { return map[string]any{"mutations": b.muts} }

// killPush pushes batches from four devices, kills the server in the middle
// and checks the rule of the protocol: a batch is one transaction (all of it
// or none of it), an acknowledged batch is there, and a replay of every batch
// with the same op ids leaves each row exactly once.
func killPush(ctx context.Context) error {
	const workers, perBatch, maxBatches = 4, 20, 400

	s, ws, err := seeded(ctx, "", 500)
	if err != nil {
		return err
	}

	defer s.Close()

	if err := s.Start(ctx); err != nil {
		return err
	}

	devices := make([]string, workers)

	for w := range workers {
		raw, err := apiDo(ctx, s, ws.Tokens[w%len(ws.Tokens)], "POST", "/api/v1/sync/devices",
			map[string]string{"name": fmt.Sprintf("drill %d", w), "platform": "other"}, map[string]string{"X-Sync-Protocol": "1"})
		if err != nil {
			return fmt.Errorf("register device: %w", err)
		}

		var d struct {
			DeviceID string `json:"device_id"`
		}

		if err := json.Unmarshal(raw, &d); err != nil || d.DeviceID == "" {
			return fmt.Errorf("register device: %s", raw)
		}

		devices[w] = d.DeviceID
	}

	var (
		mu      sync.Mutex
		batches []*batch
		wg      sync.WaitGroup
		firstOK = make(chan struct{})
		once    sync.Once
	)

	newBatch := func(w int) *batch {
		b := &batch{worker: w}

		for i := range perBatch {
			uid := qakit.ULID(time.Now())
			b.uids = append(b.uids, uid)
			b.muts = append(b.muts, map[string]any{
				"op_id": qakit.ULID(time.Now()), "entity": "transaction", "uid": uid, "op": "upsert", "base_seq": 0,
				"fields": map[string]any{
					"account_uid": ws.Accounts[i%3], "kind": "expense", "amount_cents": 100 + i,
					"description": "drill", "occurred_on": time.Now().UTC().Format("2006-01-02"),
				},
			})
		}

		return b
	}

	for w := range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range maxBatches / workers {
				b := newBatch(w)

				mu.Lock()
				batches = append(batches, b)
				mu.Unlock()

				_, err := apiDo(ctx, s, ws.Tokens[w%len(ws.Tokens)], "POST", "/api/v1/sync/push", b.payload(), syncHeaders(devices[w]))
				if err != nil {
					return // the server is gone: this batch was in flight
				}

				mu.Lock()
				b.acked = true
				mu.Unlock()

				once.Do(func() { close(firstOK) })
			}
		}()
	}

	select {
	case <-firstOK:
	case <-time.After(30 * time.Second):
		return fmt.Errorf("no push was acknowledged in 30 s")
	}

	time.Sleep(700 * time.Millisecond)
	s.Kill()
	wg.Wait()

	if err := s.Start(ctx); err != nil {
		return fmt.Errorf("restart after the kill: %w", err)
	}

	acked, inFlight, partial := 0, 0, 0

	for _, b := range batches {
		n, err := s.Scalar(ctx, "SELECT COUNT(*) FROM transactions WHERE uid IN ("+placeholders(len(b.uids))+")", anys(b.uids)...)
		if err != nil {
			return err
		}

		switch {
		case b.acked && n != perBatch:
			return fmt.Errorf("an acknowledged batch holds %d of %d rows after the restart: a lost write", n, perBatch)
		case b.acked:
			acked++
		case n != 0 && n != perBatch:
			partial++
		case n == perBatch:
			inFlight++
		}
	}

	if partial > 0 {
		return fmt.Errorf("%d batches were half applied: a batch must be all or nothing", partial)
	}

	// The devices retry what they were not sure about, with the same op ids.
	for _, b := range batches {
		raw, err := apiDo(ctx, s, ws.Tokens[b.worker%len(ws.Tokens)], "POST", "/api/v1/sync/push", b.payload(), syncHeaders(devices[b.worker]))
		if err != nil {
			return fmt.Errorf("replay: %w", err)
		}

		if bytes.Contains(raw, []byte(`"status":"rejected"`)) {
			return fmt.Errorf("replay rejected a mutation: %.200s", raw)
		}
	}

	all := 0

	for _, b := range batches {
		all += len(b.uids)

		n, err := s.Scalar(ctx, "SELECT COUNT(*) FROM transactions WHERE uid IN ("+placeholders(len(b.uids))+")", anys(b.uids)...)
		if err != nil {
			return err
		}

		if n != int64(len(b.uids)) {
			return fmt.Errorf("after the replay a batch holds %d of %d rows", n, len(b.uids))
		}
	}

	fp, err := s.Fingerprint(ctx)
	if err != nil {
		return err
	}

	if fp.Transactions != fp.DistinctUIDs {
		return fmt.Errorf("doubled rows: %d rows, %d distinct uids", fp.Transactions, fp.DistinctUIDs)
	}

	if want := int64(500 + all); fp.Transactions != want {
		return fmt.Errorf("%d transactions after the replay, want %d (500 seeded + %d pushed)", fp.Transactions, want, all)
	}

	fmt.Printf("      %d batches sent, %d acknowledged, %d in flight and applied whole, none doubled or half applied\n",
		len(batches), acked, inFlight)

	return nil
}

func syncHeaders(device string) map[string]string {
	return map[string]string{"X-Sync-Protocol": "1", "X-Device-Id": device}
}

func apiDo(ctx context.Context, s *qakit.Stack, token, method, path string, body any, headers map[string]string) ([]byte, error) {
	var rdr io.Reader

	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		rdr = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.Base+path, rdr)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode > 299 {
		return raw, fmt.Errorf("%s %s: status %d: %.160s", method, path, res.StatusCode, raw)
	}

	return raw, nil
}

func placeholders(n int) string { return strings.TrimSuffix(strings.Repeat("?,", n), ",") }

func anys(in []string) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}

	return out
}

// killImport uploads a statement of 20000 rows, confirms it and kills the
// server while the job runs. The import is all or nothing: after the restart
// the account holds none of its rows or all of them, never some, and once the
// job has finished a second run does not double them.
func killImport(ctx context.Context) error {
	const rows = 20000

	killAfter := 20 * time.Millisecond
	if v, err := time.ParseDuration(os.Getenv("DRILL_KILL_AFTER")); err == nil {
		killAfter = v
	}

	s, ws, err := seeded(ctx, "", 100)
	if err != nil {
		return err
	}

	defer s.Close()

	if err := s.Start(ctx); err != nil {
		return err
	}

	accountID, err := s.Scalar(ctx, "SELECT id FROM accounts ORDER BY id LIMIT 1")
	if err != nil {
		return err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}

	web := &http.Client{Jar: jar, Timeout: time.Minute}

	if err := expectOK(web.PostForm(s.Base+"/login", url.Values{"email": {ws.Email}, "password": {qakit.SeedPass}})); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	var csv strings.Builder

	csv.WriteString("date,description,amount\n")

	day := time.Now().UTC()
	for i := range rows {
		fmt.Fprintf(&csv, "%s,Statement line %d,-%d.%02d\n", day.AddDate(0, 0, -i%700).Format("2006-01-02"), i, 1+i%90, i%100)
	}

	var form bytes.Buffer

	mw := multipart.NewWriter(&form)
	_ = mw.WriteField("accountId", fmt.Sprint(accountID))
	part, _ := mw.CreateFormFile("file", "statement.csv")
	_, _ = part.Write([]byte(csv.String()))
	_ = mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Base+"/import", &form)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", mw.FormDataContentType())

	if err := expectOK(web.Do(req)); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	importID, err := s.Scalar(ctx, "SELECT id FROM imports ORDER BY id DESC LIMIT 1")
	if err != nil {
		return err
	}

	// The queue runs the job inside the confirm request, so the kill lands
	// while that request is in flight, after a short head start for the job.
	confirm := url.Values{"date": {"date"}, "description": {"description"}, "amount": {"amount"}, "dateFormat": {"2006-01-02"}}
	sent := make(chan struct{})

	go func() {
		defer close(sent)

		res, err := web.PostForm(fmt.Sprintf("%s/import/%d", s.Base, importID), confirm)
		if err == nil {
			_ = res.Body.Close()
		}
	}()

	time.Sleep(killAfter)
	s.Kill()
	<-sent

	midway, err := s.Fingerprint(ctx)
	if err != nil {
		return err
	}

	if added := midway.Transactions - 100; added != 0 && added != rows {
		return fmt.Errorf("the killed import left %d of %d rows: it must be all or nothing", added, rows)
	}

	if err := s.Start(ctx); err != nil {
		return fmt.Errorf("restart after the kill: %w", err)
	}

	// The job may finish after the restart; wait for a final state.
	deadline := time.Now().Add(45 * time.Second)

	for {
		fp, err := s.Fingerprint(ctx)
		if err != nil {
			return err
		}

		added := fp.Transactions - 100
		if added != 0 && added != rows {
			return fmt.Errorf("the import holds %d of %d rows after the restart", added, rows)
		}

		if fp.Transactions != fp.DistinctUIDs {
			return fmt.Errorf("doubled rows: %d rows, %d distinct uids", fp.Transactions, fp.DistinctUIDs)
		}

		status, err := statusOf(ctx, s, importID)
		if err != nil {
			return err
		}

		if status == "done" || status == "failed" {
			if status == "failed" || added != rows {
				return fmt.Errorf("the import ended %q with %d of %d rows", status, added, rows)
			}

			when := "after the job had finished"
			if midway.Transactions == 100 {
				when = "before the job committed"
			}

			fmt.Printf("      killed %s: the import ended done with %d rows, none doubled\n", when, added)

			return nil
		}

		if time.Now().After(deadline) {
			jobs, jobsErr := s.Scalar(ctx, "SELECT COUNT(*) FROM queue_jobs")

			return fmt.Errorf("the import is still %q 45 s after the restart with %d rows: the killed job was lost (queue_jobs rows: %d, %v)",
				status, added, jobs, jobsErr)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func statusOf(ctx context.Context, s *qakit.Stack, id int64) (string, error) {
	n, err := s.Scalar(ctx, "SELECT CASE status WHEN 'done' THEN 1 WHEN 'failed' THEN 2 WHEN 'queued' THEN 3 ELSE 4 END FROM imports WHERE id = ?", id)
	if err != nil {
		return "", err
	}

	return map[int64]string{1: "done", 2: "failed", 3: "queued", 4: "uploaded"}[n], nil
}

func expectOK(res *http.Response, err error) error {
	if err != nil {
		return err
	}

	defer func() { _ = res.Body.Close() }()

	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode > 299 {
		return fmt.Errorf("status %d: %.200s", res.StatusCode, raw)
	}

	return nil
}
