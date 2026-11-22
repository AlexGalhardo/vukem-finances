package ledger

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/ai/assistant"
	"github.com/AlexGalhardo/vukem-finances/app/ai/categorize"
	"github.com/AlexGalhardo/vukem-finances/app/ai/entry"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const suggestionsPath = "/suggestions"

// runTimeout bounds the on-demand categorization: at most ten model calls, and
// what finished before the deadline is kept (each batch is stored as it comes).
const runTimeout = 25 * time.Second

// AIFeatureHandlers serve the AI features that work on the person's data: the
// category suggestions, the sentence to draft and the assistant. Every route
// checks the role on the server. A suggestion or a draft needs the right to
// record; the assistant only reads, so it needs the right to read, and it only
// ever shows the asking person their own conversations.
type AIFeatureHandlers struct {
	h         *Handlers
	ai        *ai.Service
	suggest   *categorize.Service
	entry     *entry.Service
	assistant *assistant.Service
}

// NewAIFeatures builds the handlers on the shared AI service.
func NewAIFeatures(h *Handlers, svc *ai.Service, log *slog.Logger, now func() time.Time) *AIFeatureHandlers {
	return &AIFeatureHandlers{
		h: h, ai: svc,
		suggest:   categorize.New(h.Fin.DB, svc.Client, h.Fin, log, now),
		entry:     entry.New(svc.Client, h.Fin),
		assistant: assistant.New(svc.Client, h.Fin, log, now),
	}
}

// aiError turns a refusal or a failure of the AI layer into an answer with a
// safe sentence. The provider's own text never gets here.
func aiError(err error) error {
	if _, ok := errors.AsType[*errs.HTTPError](err); ok {
		return err
	}

	switch {
	case errors.Is(err, ai.ErrDisabled), errors.Is(err, ai.ErrFeatureOff), errors.Is(err, ai.ErrNoCredential):
		return errs.New(http.StatusUnprocessableEntity, "This AI feature is not turned on for the workspace.")
	case errors.Is(err, ai.ErrTooManyCalls), errors.Is(err, ai.ErrRateLimited):
		return errs.New(http.StatusTooManyRequests, "Too many requests: try again in a minute.")
	case errors.Is(err, ai.ErrMonthlyLimit):
		return errs.New(http.StatusTooManyRequests, "The monthly AI limit of the workspace is reached.")
	case errors.Is(err, ai.ErrRejected):
		return errs.New(http.StatusBadGateway, "The provider rejected the key.")
	case errors.Is(err, ai.ErrBadRequest), errors.Is(err, ai.ErrUnavailable):
		return errs.New(http.StatusBadGateway, "The provider is unavailable.")
	}

	return err
}

func (a *AIFeatureHandlers) user(c *web.Ctx) (int64, error) {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return 0, err
	}

	return u.ID, nil
}

// Suggestions renders the review list.
func (a *AIFeatureHandlers) Suggestions(c *web.Ctx) error {
	ctx := c.Context()

	pending, applied, err := a.suggest.List(ctx)
	if err != nil {
		return err
	}

	settings, err := a.ai.Settings.Get(ctx)
	if err != nil {
		return err
	}

	cur, err := a.h.WS.Current(ctx)
	if err != nil {
		return err
	}

	return pages.Suggestions.Render(c, pages.SuggestionsProps{
		Pending: pending, Applied: applied, AIOn: settings.FeatureOn(ai.FeatureCategorize),
		AutoApply: settings.AutoApplySuggestions, CanRecord: a.h.can(c, workspaces.AbilityRecord) == nil,
		Currency: cur.Currency,
	})
}

// RunSuggestions proposes categories for the uncategorized transactions now.
func (a *AIFeatureHandlers) RunSuggestions(c *web.Ctx) error {
	if err := a.h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.Context(), runTimeout)
	defer cancel()

	res, err := a.suggest.Run(ctx, categorize.Options{})
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return back(c, suggestionsPath, err)
	}

	msg := "Looked at " + strconv.Itoa(res.Considered) + " transactions: " + strconv.Itoa(res.FromRules) +
		" from your history, " + strconv.Itoa(res.FromAI) + " from AI."
	if res.AIFailed {
		msg += " AI could not finish; try again later."
	}

	return done(c, suggestionsPath, msg)
}

func (a *AIFeatureHandlers) suggestionAction(c *web.Ctx, do func(context.Context, int64) error, message string) error {
	if err := a.h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	id, err := c.ParamInt("suggestion")
	if err != nil {
		return err
	}

	if err := do(c.Context(), id); err != nil {
		return back(c, suggestionsPath, err)
	}

	if c.WantsJSON() {
		return c.JSON(http.StatusOK, map[string]any{"ok": true})
	}

	return done(c, suggestionsPath, message)
}

// AcceptSuggestion applies one suggestion.
func (a *AIFeatureHandlers) AcceptSuggestion(c *web.Ctx) error {
	return a.suggestionAction(c, a.suggest.Accept, "Category applied.")
}

// RejectSuggestion dismisses one suggestion.
func (a *AIFeatureHandlers) RejectSuggestion(c *web.Ctx) error {
	return a.suggestionAction(c, a.suggest.Reject, "Suggestion dismissed.")
}

// UndoSuggestion reverses an auto-applied suggestion within seven days.
func (a *AIFeatureHandlers) UndoSuggestion(c *web.Ctx) error {
	return a.suggestionAction(c, a.suggest.Undo, "Category removed.")
}

// AcceptAllSuggestions applies every pending suggestion of at least the given confidence.
func (a *AIFeatureHandlers) AcceptAllSuggestions(c *web.Ctx) error {
	if err := a.h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.AcceptAllSchema)
	if err != nil {
		return err
	}

	n, err := a.suggest.AcceptAll(c.Context(), int(id(in.Min)))
	if err != nil {
		return back(c, suggestionsPath, err)
	}

	if c.WantsJSON() {
		return c.JSON(http.StatusOK, map[string]any{"accepted": n})
	}

	return done(c, suggestionsPath, strconv.Itoa(n)+" categories applied.")
}

// DraftEntry reads a sentence into the draft of a transaction. Nothing is
// saved: the draft opens in the transaction dialog and the person confirms.
func (a *AIFeatureHandlers) DraftEntry(c *web.Ctx) error {
	if err := a.h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.EntryDraftSchema)
	if err != nil {
		return err
	}

	draft, err := a.entry.Parse(c.Context(), in.Text)
	if err != nil {
		return aiError(err)
	}

	return c.JSON(http.StatusOK, draft)
}

// DraftStatus tells the transaction dialog whether to offer the sentence field.
func (a *AIFeatureHandlers) DraftStatus(c *web.Ctx) error {
	settings, err := a.ai.Settings.Get(c.Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"enabled":   settings.FeatureOn(ai.FeatureCategorize) && a.h.can(c, workspaces.AbilityRecord) == nil,
		"textLimit": entry.TextMax,
	})
}

// AssistantStatus tells the dialog whether the assistant is on and lists the
// conversations of the person asking.
func (a *AIFeatureHandlers) AssistantStatus(c *web.Ctx) error {
	ctx := c.Context()

	settings, err := a.ai.Settings.Get(ctx)
	if err != nil {
		return err
	}

	uid, err := a.user(c)
	if err != nil {
		return err
	}

	list, err := a.assistant.Conversations(ctx, uid)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"enabled": settings.FeatureOn(ai.FeatureAssistant), "conversations": list,
		"questionMax": assistant.QuestionMax,
	})
}

// AssistantAsk answers one question. It needs transactions.record: every
// question spends the monthly AI limit the owner pays for, and a viewer changes
// nothing in the workspace, that budget included.
func (a *AIFeatureHandlers) AssistantAsk(c *web.Ctx) error {
	if err := a.h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.AssistantAskSchema)
	if err != nil {
		return err
	}

	uid, err := a.user(c)
	if err != nil {
		return err
	}

	reply, err := a.assistant.Ask(c.Context(), uid, id(in.ConversationID), in.Question)
	if err != nil {
		return aiError(err)
	}

	return c.JSON(http.StatusOK, reply)
}

// AssistantMessages returns the turns of one of the caller's conversations. A
// viewer cannot start one, so an id that is not theirs reads as empty; someone
// demoted to viewer still reads what they asked before.
func (a *AIFeatureHandlers) AssistantMessages(c *web.Ctx) error {
	cid, err := c.ParamInt("conversation")
	if err != nil {
		return err
	}

	uid, err := a.user(c)
	if err != nil {
		return err
	}

	msgs, err := a.assistant.Messages(c.Context(), uid, cid)
	if errors.Is(err, orm.ErrNotFound) && a.h.can(c, workspaces.AbilityRecord) != nil {
		return c.JSON(http.StatusOK, map[string]any{"messages": []assistant.MessageView{}})
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{"messages": msgs})
}

// AssistantDelete deletes one of the caller's conversations. Anyone can erase
// what they asked, a person demoted to viewer included; a viewer who names
// anything else is refused like on every other write.
func (a *AIFeatureHandlers) AssistantDelete(c *web.Ctx) error {
	cid, err := c.ParamInt("conversation")
	if err != nil {
		return err
	}

	uid, err := a.user(c)
	if err != nil {
		return err
	}

	if err := a.assistant.Delete(c.Context(), uid, cid); err != nil {
		if errors.Is(err, orm.ErrNotFound) {
			if denied := a.h.can(c, workspaces.AbilityRecord); denied != nil {
				return denied
			}
		}

		return err
	}

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

// AssistantDeleteAll deletes every conversation of the caller in the workspace.
// A viewer with nothing of their own to erase is refused.
func (a *AIFeatureHandlers) AssistantDeleteAll(c *web.Ctx) error {
	uid, err := a.user(c)
	if err != nil {
		return err
	}

	n, err := a.assistant.DeleteAll(c.Context(), uid)
	if err != nil {
		return err
	}

	if n == 0 {
		if denied := a.h.can(c, workspaces.AbilityRecord); denied != nil {
			return denied
		}
	}

	return c.JSON(http.StatusOK, map[string]any{"deleted": n})
}
