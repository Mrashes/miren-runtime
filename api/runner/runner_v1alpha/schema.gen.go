package runner_v1alpha

import (
	"time"

	entity "miren.dev/runtime/pkg/entity"
	schema "miren.dev/runtime/pkg/entity/schema"
	types "miren.dev/runtime/pkg/entity/types"
)

const (
	RunnerInviteClaimedAtId     = entity.Id("dev.miren.runner/runner_invite.claimed_at")
	RunnerInviteClaimedById     = entity.Id("dev.miren.runner/runner_invite.claimed_by")
	RunnerInviteCodeHashId      = entity.Id("dev.miren.runner/runner_invite.code_hash")
	RunnerInviteCreatedAtId     = entity.Id("dev.miren.runner/runner_invite.created_at")
	RunnerInviteExpiresAtId     = entity.Id("dev.miren.runner/runner_invite.expires_at")
	RunnerInviteLabelsId        = entity.Id("dev.miren.runner/runner_invite.labels")
	RunnerInviteStatusId        = entity.Id("dev.miren.runner/runner_invite.status")
	RunnerInviteStatusPendingId = entity.Id("dev.miren.runner/status.pending")
	RunnerInviteStatusClaimedId = entity.Id("dev.miren.runner/status.claimed")
	RunnerInviteStatusRevokedId = entity.Id("dev.miren.runner/status.revoked")
	RunnerInviteStatusExpiredId = entity.Id("dev.miren.runner/status.expired")
)

type RunnerInvite struct {
	ID        entity.Id          `json:"id"`
	ClaimedAt time.Time          `cbor:"claimed_at,omitempty" json:"claimed_at,omitempty"`
	ClaimedBy string             `cbor:"claimed_by,omitempty" json:"claimed_by,omitempty"`
	CodeHash  string             `cbor:"code_hash,omitempty" json:"code_hash,omitempty"`
	CreatedAt time.Time          `cbor:"created_at,omitempty" json:"created_at,omitempty"`
	ExpiresAt time.Time          `cbor:"expires_at,omitempty" json:"expires_at,omitempty"`
	Labels    types.Labels       `cbor:"labels,omitempty" json:"labels,omitempty"`
	Status    RunnerInviteStatus `cbor:"status,omitempty" json:"status,omitempty"`
}

type RunnerInviteStatus string

const (
	PENDING RunnerInviteStatus = "status.pending"
	CLAIMED RunnerInviteStatus = "status.claimed"
	REVOKED RunnerInviteStatus = "status.revoked"
	EXPIRED RunnerInviteStatus = "status.expired"
)

var runner_invitestatusFromId = map[entity.Id]RunnerInviteStatus{RunnerInviteStatusPendingId: PENDING, RunnerInviteStatusClaimedId: CLAIMED, RunnerInviteStatusRevokedId: REVOKED, RunnerInviteStatusExpiredId: EXPIRED}
var runner_invitestatusToId = map[RunnerInviteStatus]entity.Id{PENDING: RunnerInviteStatusPendingId, CLAIMED: RunnerInviteStatusClaimedId, REVOKED: RunnerInviteStatusRevokedId, EXPIRED: RunnerInviteStatusExpiredId}

func (o *RunnerInvite) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(RunnerInviteClaimedAtId); ok && a.Value.Kind() == entity.KindTime {
		o.ClaimedAt = a.Value.Time()
	}
	if a, ok := e.Get(RunnerInviteClaimedById); ok && a.Value.Kind() == entity.KindString {
		o.ClaimedBy = a.Value.String()
	}
	if a, ok := e.Get(RunnerInviteCodeHashId); ok && a.Value.Kind() == entity.KindString {
		o.CodeHash = a.Value.String()
	}
	if a, ok := e.Get(RunnerInviteCreatedAtId); ok && a.Value.Kind() == entity.KindTime {
		o.CreatedAt = a.Value.Time()
	}
	if a, ok := e.Get(RunnerInviteExpiresAtId); ok && a.Value.Kind() == entity.KindTime {
		o.ExpiresAt = a.Value.Time()
	}
	for _, a := range e.GetAll(RunnerInviteLabelsId) {
		if a.Value.Kind() == entity.KindLabel {
			o.Labels = append(o.Labels, a.Value.Label())
		}
	}
	if a, ok := e.Get(RunnerInviteStatusId); ok && a.Value.Kind() == entity.KindId {
		o.Status = runner_invitestatusFromId[a.Value.Id()]
	}
}

func (o *RunnerInvite) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindRunnerInvite)
}

func (o *RunnerInvite) ShortKind() string {
	return "runner_invite"
}

func (o *RunnerInvite) Kind() entity.Id {
	return KindRunnerInvite
}

func (o *RunnerInvite) EntityId() entity.Id {
	return o.ID
}

func (o *RunnerInvite) Encode() (attrs []entity.Attr) {
	if !entity.Empty(o.ClaimedAt) {
		attrs = append(attrs, entity.Time(RunnerInviteClaimedAtId, o.ClaimedAt))
	}
	if !entity.Empty(o.ClaimedBy) {
		attrs = append(attrs, entity.String(RunnerInviteClaimedById, o.ClaimedBy))
	}
	if !entity.Empty(o.CodeHash) {
		attrs = append(attrs, entity.String(RunnerInviteCodeHashId, o.CodeHash))
	}
	if !entity.Empty(o.CreatedAt) {
		attrs = append(attrs, entity.Time(RunnerInviteCreatedAtId, o.CreatedAt))
	}
	if !entity.Empty(o.ExpiresAt) {
		attrs = append(attrs, entity.Time(RunnerInviteExpiresAtId, o.ExpiresAt))
	}
	for _, v := range o.Labels {
		attrs = append(attrs, entity.Label(RunnerInviteLabelsId, v.Key, v.Value))
	}
	if a, ok := runner_invitestatusToId[o.Status]; ok {
		attrs = append(attrs, entity.Ref(RunnerInviteStatusId, a))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindRunnerInvite))
	return
}

func (o *RunnerInvite) Empty() bool {
	if !entity.Empty(o.ClaimedAt) {
		return false
	}
	if !entity.Empty(o.ClaimedBy) {
		return false
	}
	if !entity.Empty(o.CodeHash) {
		return false
	}
	if !entity.Empty(o.CreatedAt) {
		return false
	}
	if !entity.Empty(o.ExpiresAt) {
		return false
	}
	if len(o.Labels) != 0 {
		return false
	}
	if o.Status != "" {
		return false
	}
	return true
}

func (o *RunnerInvite) InitSchema(sb *schema.SchemaBuilder) {
	sb.Time("claimed_at", "dev.miren.runner/runner_invite.claimed_at", schema.Doc("When the invite was claimed"))
	sb.String("claimed_by", "dev.miren.runner/runner_invite.claimed_by", schema.Doc("Runner ID that claimed this invite"))
	sb.String("code_hash", "dev.miren.runner/runner_invite.code_hash", schema.Doc("SHA-256 hash of the join code (code itself is not stored)"), schema.Indexed)
	sb.Time("created_at", "dev.miren.runner/runner_invite.created_at", schema.Doc("When the invite was created"))
	sb.Time("expires_at", "dev.miren.runner/runner_invite.expires_at", schema.Doc("When the invite expires"))
	sb.Label("labels", "dev.miren.runner/runner_invite.labels", schema.Doc("Labels to apply to the runner when it joins"), schema.Many)
	sb.Singleton("dev.miren.runner/status.pending")
	sb.Singleton("dev.miren.runner/status.claimed")
	sb.Singleton("dev.miren.runner/status.revoked")
	sb.Singleton("dev.miren.runner/status.expired")
	sb.Ref("status", "dev.miren.runner/runner_invite.status", schema.Doc("Status of the invite"), schema.Indexed, schema.Choices(RunnerInviteStatusPendingId, RunnerInviteStatusClaimedId, RunnerInviteStatusRevokedId, RunnerInviteStatusExpiredId))
}

var (
	KindRunnerInvite = entity.Id("dev.miren.runner/kind.runner_invite")
	Schema           = entity.Id("dev.miren.runner/schema.v1alpha")
)

func init() {
	schema.Register("dev.miren.runner", "v1alpha", func(sb *schema.SchemaBuilder) {
		(&RunnerInvite{}).InitSchema(sb)
	})
	schema.RegisterEncodedSchema("dev.miren.runner", "v1alpha", []byte("\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x94\xd3_n\xbc \x10\a\xf0\x9b\xfc~\x0fM\x93\xf6e\x9b\x9e\x88\x8ceĩ0\x12@\xa2'\xe89\x9a\x9aް}nD\xba\xeb\x9fTw_\f#_?0A\x06\xc9`\xd0J\x8c'C\x0e\xf9\xe4ZftX\x13K\xff\xdeݭ'\x9eƉ<\x16đ\x02~&\xa2\xfb\xb7\x89.R\x93\xf8]\xca\xc6\x00\xf1f\xc1\xb2$\xd4ҿ}\x14$\xbb\xc7}\xea\xf4\xa2\x81\fJ\x01!-\xfd:\xabCoQ\x062x\x13T\xf4K\xa8\xe8\x13T\xfa\xe0\x88U\xa2\x1e\x8e\xa8F\xa2\xa8\xc0WI\xa2K\xb9\x86\x0e\xf7\xe4\x10¼\xb9K}[s\xd8Yr\xe8\xcfЬ>C\xc3\b\xdd\x1f@\x1a\n\xd4^\x1a\xe0\xfe+Qe~32\x98\xc6W9>@h\xfd$\xe4q\xda\brk\xea\xf1!\"\xe8\x16\xfd\xa0\xf29t\xff7\xe2\xf4\xdd\xef\xc1\xa9\xa9\xa7\x9d`\x0e(\x8b,\x89\xd5\xdf\xc1\x1cP\x0ecS\xef\x899\xa0\":O\r\xab\xf8\f\xdaV\xa0\xad#\x03\xae\x17\xe3\x7fn\x16\x8d\xaf\xa3\xb5\xaf\x1a\x17\xc4tŖѫ.\xdc\x0f\x00\x00\x00\xff\xff\x01\x00\x00\xff\xff\xf8\b\x12W\xb4\x03\x00\x00"))
}
