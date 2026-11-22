package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The planted values are assembled at run time so this file holds nothing a
// secret scanner would flag.
func planted() map[string]string {
	rnd := strings.Repeat("aB3dE5gH", 4)

	return map[string]string{
		"bearer token":         "Authorization: Bearer " + rnd,
		"api token":            "token " + "vf" + "_" + "7Qx9Lm2" + "." + rnd,
		"jwt":                  "access " + "ey" + "Jhbg" + "ciOiJIUzI1" + "." + "eyJzdWIiOiIx" + "NDIifQ" + "." + "c2lnbmF0dXJl" + "x",
		"payment provider key": "key=" + "sk" + "_" + "test" + "_" + rnd,
		"provider api key":     "using " + "sk" + "-" + rnd,
		"bot token":            "url https://api.telegram.org/bot" + "1234567" + ":" + rnd[:30] + "/getMe",
		"app key":              "APP_KEY=" + "base" + "64:" + strings.Repeat("Ab1/", 11),
		"credential field":     `{"pass` + `word":"hunter2hunter2"}`,
		"session id":           "Cookie: vukem" + "_session=" + rnd,
		"card number":          "paid with " + "4" + "242 4242 4242 4242" + " today",
	}
}

func TestScanCatchesEveryPlantedSecret(t *testing.T) {
	t.Parallel()

	for rule, line := range planted() {
		found, err := NewScanner(nil).Scan("planted", strings.NewReader("ok line\n"+line+"\n"))
		if err != nil {
			t.Fatal(err)
		}

		var hit bool

		for _, f := range found {
			hit = hit || f.Rule == rule
			if f.Line != 2 {
				t.Errorf("%s: finding on line %d, want 2", rule, f.Line)
			}
		}

		if !hit {
			t.Errorf("%s: not caught in %q (found %v)", rule, line, found)
		}
	}
}

func TestScanNeverPrintsTheSecret(t *testing.T) {
	t.Parallel()

	for rule, line := range planted() {
		found, _ := NewScanner(nil).Scan("planted", strings.NewReader(line))
		for _, f := range found {
			if strings.Contains(f.String(), line[len(line)/2:]) {
				t.Errorf("%s: the finding repeats the secret: %q", rule, f)
			}
		}
	}
}

func TestScanKnownSecretIsCaughtVerbatim(t *testing.T) {
	t.Parallel()

	secret := "placeholder-" + "webhook-value"
	found, _ := NewScanner([]string{secret, "short"}).Scan("log", strings.NewReader("hello short\nGET /x/"+secret+"\n"))

	if len(found) != 1 || found[0].Rule != "known secret" || found[0].Line != 2 {
		t.Errorf("found = %v, want one known secret on line 2", found)
	}
}

func TestScanLeavesRedactedAndOrdinaryLinesAlone(t *testing.T) {
	t.Parallel()

	log := strings.Join([]string{
		`time=2026-11-22T15:30:29.336-03:00 level=INFO msg="request" path=/abacatepay/webhook/{secret} status=200`,
		`Authorization: Bearer [redacted]`,
		`{"password":"[redacted]","apiSecret":"***"}`,
		`token=[redacted] ts=1795377029336 id=01J0000000000000000000000A`,
		`bot url https://api.telegram.org/bot[redacted]/getMe`,
		`amount_cents=4242424242424 total=1234567890123`,
	}, "\n")

	found, err := NewScanner(nil).Scan("log", strings.NewReader(log))
	if err != nil || len(found) != 0 {
		t.Errorf("found = %v (%v), want nothing", found, err)
	}
}

func TestRunExitCodes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	clean := filepath.Join(dir, "clean.log")
	dirty := filepath.Join(dir, "dirty.log")

	if err := os.WriteFile(clean, []byte("level=INFO msg=ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(dirty, []byte(planted()["bearer token"]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer

	if code := run([]string{clean}, nil, &out, &errOut); code != 0 {
		t.Errorf("clean: code = %d, output %s", code, out.String())
	}

	out.Reset()

	if code := run([]string{dir}, nil, &out, &errOut); code != 1 || !strings.Contains(out.String(), "bearer token") {
		t.Errorf("dirty: code = %d, output %s", code, out.String())
	}

	if code := run([]string{"-"}, strings.NewReader(planted()["card number"]), &out, &errOut); code != 1 {
		t.Errorf("stdin: code = %d", code)
	}

	if code := run(nil, nil, &out, &errOut); code != 2 {
		t.Errorf("no input: code = %d, want 2", code)
	}

	if code := run([]string{filepath.Join(dir, "missing")}, nil, &out, &errOut); code != 2 {
		t.Errorf("missing file: code = %d, want 2", code)
	}
}
