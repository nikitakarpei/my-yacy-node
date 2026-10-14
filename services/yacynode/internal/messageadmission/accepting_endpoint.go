package messageadmission

import (
	"context"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	subjectSizeLimit = 100
	bodySizeLimit    = 10240
)

const (
	refusalForeignSender  = "seed names another peer than iam"
	refusalEmptyText      = "subject or body is empty"
	refusalOversizedText  = "subject or body is larger than announced"
	refusalSpentAllowance = "the node or the sender has no messages left this hour"
)

type acceptingEndpoint struct {
	identity  nodeidentity.Identity
	allowance *messageAllowance
	inbox     Inbox
}

func (e acceptingEndpoint) Serve(
	ctx context.Context,
	req yacyproto.MessageRequest,
) (yacyproto.MessageResponse, error) {
	switch {
	case !e.identity.Addresses(req.NetworkName, req.YouAre):
		return refusalOfForeignRequest(ctx), nil
	case req.Process == yacyproto.ProcessPermission:
		return answerOf(yacyproto.MessageResponseWelcome), nil
	default:
		return e.deliveryAnswerTo(ctx, req), nil
	}
}

func (e acceptingEndpoint) deliveryAnswerTo(
	ctx context.Context,
	req yacyproto.MessageRequest,
) yacyproto.MessageResponse {
	message, refusal := e.admissionOf(ctx, req)
	if refusal != "" {
		slog.WarnContext(ctx, "message refused",
			slog.String("reason", refusal),
			slog.String("sender", httpguard.RemoteAddr(ctx)),
		)

		return answerOf(yacyproto.MessageResponseRefused)
	}

	if !e.inbox.Deliver(ctx, message) {
		return answerOf(yacyproto.MessageResponseRefused)
	}

	slog.DebugContext(ctx, "message delivered",
		slog.String("sender", message.Sender.Hash.String()),
	)

	return answerOf(yacyproto.MessageResponseAccepted)
}

func (e acceptingEndpoint) admissionOf(
	ctx context.Context,
	req yacyproto.MessageRequest,
) (Message, string) {
	seed, carried := req.MySeed.Get()
	subject := strings.TrimSpace(req.Subject)
	body := strings.TrimSpace(req.Body)
	address := httpguard.RemoteAddr(ctx)

	switch {
	case !carried || seed.Hash != req.Iam:
		return Message{}, refusalForeignSender
	case subject == "" || body == "":
		return Message{}, refusalEmptyText
	case utf8.RuneCountInString(subject) > subjectSizeLimit ||
		utf8.RuneCountInString(body) > bodySizeLimit:
		return Message{}, refusalOversizedText
	case !e.allowance.take(address):
		return Message{}, refusalSpentAllowance
	}

	return Message{
		Subject: subject,
		Body:    body,
		Sender: Sender{
			Hash:    seed.Hash,
			Name:    seed.Name.String(),
			Address: address,
		},
	}, ""
}

func answerOf(response string) yacyproto.MessageResponse {
	return yacyproto.MessageResponse{
		Response:    response,
		MessageSize: bodySizeLimit,
	}
}

func refusalOfForeignRequest(ctx context.Context) yacyproto.MessageResponse {
	slog.DebugContext(ctx, "message refused, request names another peer or network")

	return yacyproto.MessageResponse{Response: yacyproto.MessageResponseRefused}
}
