package ingress_v1alpha

import (
	entity "miren.dev/runtime/pkg/entity"
	schema "miren.dev/runtime/pkg/entity/schema"
)

const (
	HttpRouteAppId              = entity.Id("dev.miren.ingress/http_route.app")
	HttpRouteClaimMappingsId    = entity.Id("dev.miren.ingress/http_route.claim_mappings")
	HttpRouteDefaultId          = entity.Id("dev.miren.ingress/http_route.default")
	HttpRouteHostId             = entity.Id("dev.miren.ingress/http_route.host")
	HttpRouteOidcProviderId     = entity.Id("dev.miren.ingress/http_route.oidc_provider")
	HttpRoutePasswordProviderId = entity.Id("dev.miren.ingress/http_route.password_provider")
	HttpRouteWafProfileId       = entity.Id("dev.miren.ingress/http_route.waf_profile")
)

type HttpRoute struct {
	ID               entity.Id       `json:"id"`
	App              entity.Id       `cbor:"app,omitempty" json:"app,omitempty"`
	ClaimMappings    []ClaimMappings `cbor:"claim_mappings,omitempty" json:"claim_mappings,omitempty"`
	Default          bool            `cbor:"default,omitempty" json:"default,omitempty"`
	Host             string          `cbor:"host,omitempty" json:"host,omitempty"`
	OidcProvider     entity.Id       `cbor:"oidc_provider,omitempty" json:"oidc_provider,omitempty"`
	PasswordProvider entity.Id       `cbor:"password_provider,omitempty" json:"password_provider,omitempty"`
	WafProfile       entity.Id       `cbor:"waf_profile,omitempty" json:"waf_profile,omitempty"`
}

func (o *HttpRoute) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(HttpRouteAppId); ok && a.Value.Kind() == entity.KindId {
		o.App = a.Value.Id()
	}
	for _, a := range e.GetAll(HttpRouteClaimMappingsId) {
		if a.Value.Kind() == entity.KindComponent {
			var v ClaimMappings
			v.Decode(a.Value.Component())
			o.ClaimMappings = append(o.ClaimMappings, v)
		}
	}
	if a, ok := e.Get(HttpRouteDefaultId); ok && a.Value.Kind() == entity.KindBool {
		o.Default = a.Value.Bool()
	}
	if a, ok := e.Get(HttpRouteHostId); ok && a.Value.Kind() == entity.KindString {
		o.Host = a.Value.String()
	}
	if a, ok := e.Get(HttpRouteOidcProviderId); ok && a.Value.Kind() == entity.KindId {
		o.OidcProvider = a.Value.Id()
	}
	if a, ok := e.Get(HttpRoutePasswordProviderId); ok && a.Value.Kind() == entity.KindId {
		o.PasswordProvider = a.Value.Id()
	}
	if a, ok := e.Get(HttpRouteWafProfileId); ok && a.Value.Kind() == entity.KindId {
		o.WafProfile = a.Value.Id()
	}
}

func (o *HttpRoute) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindHttpRoute)
}

func (o *HttpRoute) ShortKind() string {
	return "http_route"
}

func (o *HttpRoute) Kind() entity.Id {
	return KindHttpRoute
}

func (o *HttpRoute) EntityId() entity.Id {
	return o.ID
}

func (o *HttpRoute) Encode() (attrs []entity.Attr) {
	if !entity.Empty(o.App) {
		attrs = append(attrs, entity.Ref(HttpRouteAppId, o.App))
	}
	for _, v := range o.ClaimMappings {
		attrs = append(attrs, entity.Component(HttpRouteClaimMappingsId, v.Encode()))
	}
	attrs = append(attrs, entity.Bool(HttpRouteDefaultId, o.Default))
	if !entity.Empty(o.Host) {
		attrs = append(attrs, entity.String(HttpRouteHostId, o.Host))
	}
	if !entity.Empty(o.OidcProvider) {
		attrs = append(attrs, entity.Ref(HttpRouteOidcProviderId, o.OidcProvider))
	}
	if !entity.Empty(o.PasswordProvider) {
		attrs = append(attrs, entity.Ref(HttpRoutePasswordProviderId, o.PasswordProvider))
	}
	if !entity.Empty(o.WafProfile) {
		attrs = append(attrs, entity.Ref(HttpRouteWafProfileId, o.WafProfile))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindHttpRoute))
	return
}

func (o *HttpRoute) Empty() bool {
	if !entity.Empty(o.App) {
		return false
	}
	if len(o.ClaimMappings) != 0 {
		return false
	}
	if !entity.Empty(o.Default) {
		return false
	}
	if !entity.Empty(o.Host) {
		return false
	}
	if !entity.Empty(o.OidcProvider) {
		return false
	}
	if !entity.Empty(o.PasswordProvider) {
		return false
	}
	if !entity.Empty(o.WafProfile) {
		return false
	}
	return true
}

func (o *HttpRoute) InitSchema(sb *schema.SchemaBuilder) {
	sb.Ref("app", "dev.miren.ingress/http_route.app", schema.Doc("The application to route to"), schema.Indexed, schema.Tags("dev.miren.app_ref"))
	sb.Component("claim_mappings", "dev.miren.ingress/http_route.claim_mappings", schema.Doc("Mappings from JWT claims to HTTP headers"), schema.Many)
	(&ClaimMappings{}).InitSchema(sb.Builder("http_route.claim_mappings"))
	sb.Bool("default", "dev.miren.ingress/http_route.default", schema.Doc("Whether this is the default route for routing"), schema.Indexed)
	sb.String("host", "dev.miren.ingress/http_route.host", schema.Doc("The hostname to match on for the application"), schema.Indexed)
	sb.Ref("oidc_provider", "dev.miren.ingress/http_route.oidc_provider", schema.Doc("Reference to an OIDC provider for authentication"), schema.Indexed)
	sb.Ref("password_provider", "dev.miren.ingress/http_route.password_provider", schema.Doc("Reference to a password provider for simple authentication"), schema.Indexed)
	sb.Ref("waf_profile", "dev.miren.ingress/http_route.waf_profile", schema.Doc("Reference to a WAF profile for request filtering"))
}

const (
	ClaimMappingsClaimId  = entity.Id("dev.miren.ingress/claim_mappings.claim")
	ClaimMappingsHeaderId = entity.Id("dev.miren.ingress/claim_mappings.header")
)

type ClaimMappings struct {
	Claim  string `cbor:"claim,omitempty" json:"claim,omitempty"`
	Header string `cbor:"header,omitempty" json:"header,omitempty"`
}

func (o *ClaimMappings) Decode(e entity.AttrGetter) {
	if a, ok := e.Get(ClaimMappingsClaimId); ok && a.Value.Kind() == entity.KindString {
		o.Claim = a.Value.String()
	}
	if a, ok := e.Get(ClaimMappingsHeaderId); ok && a.Value.Kind() == entity.KindString {
		o.Header = a.Value.String()
	}
}

func (o *ClaimMappings) Encode() (attrs []entity.Attr) {
	if !entity.Empty(o.Claim) {
		attrs = append(attrs, entity.String(ClaimMappingsClaimId, o.Claim))
	}
	if !entity.Empty(o.Header) {
		attrs = append(attrs, entity.String(ClaimMappingsHeaderId, o.Header))
	}
	return
}

func (o *ClaimMappings) Empty() bool {
	if !entity.Empty(o.Claim) {
		return false
	}
	if !entity.Empty(o.Header) {
		return false
	}
	return true
}

func (o *ClaimMappings) InitSchema(sb *schema.SchemaBuilder) {
	sb.String("claim", "dev.miren.ingress/claim_mappings.claim", schema.Doc("The JWT claim name (e.g. email, sub, name)"))
	sb.String("header", "dev.miren.ingress/claim_mappings.header", schema.Doc("The HTTP header name to inject (e.g. X-User-Email)"))
}

const (
	OidcProviderClientIdId     = entity.Id("dev.miren.ingress/oidc_provider.client_id")
	OidcProviderClientSecretId = entity.Id("dev.miren.ingress/oidc_provider.client_secret")
	OidcProviderNameId         = entity.Id("dev.miren.ingress/oidc_provider.name")
	OidcProviderProviderUrlId  = entity.Id("dev.miren.ingress/oidc_provider.provider_url")
	OidcProviderScopesId       = entity.Id("dev.miren.ingress/oidc_provider.scopes")
)

type OidcProvider struct {
	ID           entity.Id `json:"id"`
	ClientId     string    `cbor:"client_id,omitempty" json:"client_id,omitempty"`
	ClientSecret string    `cbor:"client_secret,omitempty" json:"client_secret,omitempty"`
	Name         string    `cbor:"name,omitempty" json:"name,omitempty"`
	ProviderUrl  string    `cbor:"provider_url,omitempty" json:"provider_url,omitempty"`
	Scopes       string    `cbor:"scopes,omitempty" json:"scopes,omitempty"`
}

func (o *OidcProvider) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(OidcProviderClientIdId); ok && a.Value.Kind() == entity.KindString {
		o.ClientId = a.Value.String()
	}
	if a, ok := e.Get(OidcProviderClientSecretId); ok && a.Value.Kind() == entity.KindString {
		o.ClientSecret = a.Value.String()
	}
	if a, ok := e.Get(OidcProviderNameId); ok && a.Value.Kind() == entity.KindString {
		o.Name = a.Value.String()
	}
	if a, ok := e.Get(OidcProviderProviderUrlId); ok && a.Value.Kind() == entity.KindString {
		o.ProviderUrl = a.Value.String()
	}
	if a, ok := e.Get(OidcProviderScopesId); ok && a.Value.Kind() == entity.KindString {
		o.Scopes = a.Value.String()
	}
}

func (o *OidcProvider) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindOidcProvider)
}

func (o *OidcProvider) ShortKind() string {
	return "oidc_provider"
}

func (o *OidcProvider) Kind() entity.Id {
	return KindOidcProvider
}

func (o *OidcProvider) EntityId() entity.Id {
	return o.ID
}

func (o *OidcProvider) Encode() (attrs []entity.Attr) {
	if !entity.Empty(o.ClientId) {
		attrs = append(attrs, entity.String(OidcProviderClientIdId, o.ClientId))
	}
	if !entity.Empty(o.ClientSecret) {
		attrs = append(attrs, entity.String(OidcProviderClientSecretId, o.ClientSecret))
	}
	if !entity.Empty(o.Name) {
		attrs = append(attrs, entity.String(OidcProviderNameId, o.Name))
	}
	if !entity.Empty(o.ProviderUrl) {
		attrs = append(attrs, entity.String(OidcProviderProviderUrlId, o.ProviderUrl))
	}
	if !entity.Empty(o.Scopes) {
		attrs = append(attrs, entity.String(OidcProviderScopesId, o.Scopes))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindOidcProvider))
	return
}

func (o *OidcProvider) Empty() bool {
	if !entity.Empty(o.ClientId) {
		return false
	}
	if !entity.Empty(o.ClientSecret) {
		return false
	}
	if !entity.Empty(o.Name) {
		return false
	}
	if !entity.Empty(o.ProviderUrl) {
		return false
	}
	if !entity.Empty(o.Scopes) {
		return false
	}
	return true
}

func (o *OidcProvider) InitSchema(sb *schema.SchemaBuilder) {
	sb.String("client_id", "dev.miren.ingress/oidc_provider.client_id", schema.Doc("The OAuth2 client ID"))
	sb.String("client_secret", "dev.miren.ingress/oidc_provider.client_secret", schema.Doc("The OAuth2 client secret"))
	sb.String("name", "dev.miren.ingress/oidc_provider.name", schema.Doc("A unique name for this OIDC provider"), schema.Indexed)
	sb.String("provider_url", "dev.miren.ingress/oidc_provider.provider_url", schema.Doc("The OIDC provider URL (e.g. https://accounts.google.com)"), schema.Indexed)
	sb.String("scopes", "dev.miren.ingress/oidc_provider.scopes", schema.Doc("Space-separated list of OAuth2 scopes (e.g. \"openid email profile\")"))
}

const (
	PasswordProviderNameId         = entity.Id("dev.miren.ingress/password_provider.name")
	PasswordProviderPasswordHashId = entity.Id("dev.miren.ingress/password_provider.password_hash")
)

type PasswordProvider struct {
	ID           entity.Id `json:"id"`
	Name         string    `cbor:"name,omitempty" json:"name,omitempty"`
	PasswordHash string    `cbor:"password_hash,omitempty" json:"password_hash,omitempty"`
}

func (o *PasswordProvider) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(PasswordProviderNameId); ok && a.Value.Kind() == entity.KindString {
		o.Name = a.Value.String()
	}
	if a, ok := e.Get(PasswordProviderPasswordHashId); ok && a.Value.Kind() == entity.KindString {
		o.PasswordHash = a.Value.String()
	}
}

func (o *PasswordProvider) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindPasswordProvider)
}

func (o *PasswordProvider) ShortKind() string {
	return "password_provider"
}

func (o *PasswordProvider) Kind() entity.Id {
	return KindPasswordProvider
}

func (o *PasswordProvider) EntityId() entity.Id {
	return o.ID
}

func (o *PasswordProvider) Encode() (attrs []entity.Attr) {
	if !entity.Empty(o.Name) {
		attrs = append(attrs, entity.String(PasswordProviderNameId, o.Name))
	}
	if !entity.Empty(o.PasswordHash) {
		attrs = append(attrs, entity.String(PasswordProviderPasswordHashId, o.PasswordHash))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindPasswordProvider))
	return
}

func (o *PasswordProvider) Empty() bool {
	if !entity.Empty(o.Name) {
		return false
	}
	if !entity.Empty(o.PasswordHash) {
		return false
	}
	return true
}

func (o *PasswordProvider) InitSchema(sb *schema.SchemaBuilder) {
	sb.String("name", "dev.miren.ingress/password_provider.name", schema.Doc("A unique name for this password provider"), schema.Indexed)
	sb.String("password_hash", "dev.miren.ingress/password_provider.password_hash", schema.Doc("bcrypt hash of the shared password"))
}

const (
	WafProfileParanoiaLevelId = entity.Id("dev.miren.ingress/waf_profile.paranoia_level")
)

type WafProfile struct {
	ID            entity.Id `json:"id"`
	ParanoiaLevel int64     `cbor:"paranoia_level,omitempty" json:"paranoia_level,omitempty"`
}

func (o *WafProfile) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(WafProfileParanoiaLevelId); ok && a.Value.Kind() == entity.KindInt64 {
		o.ParanoiaLevel = a.Value.Int64()
	}
}

func (o *WafProfile) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindWafProfile)
}

func (o *WafProfile) ShortKind() string {
	return "waf_profile"
}

func (o *WafProfile) Kind() entity.Id {
	return KindWafProfile
}

func (o *WafProfile) EntityId() entity.Id {
	return o.ID
}

func (o *WafProfile) Encode() (attrs []entity.Attr) {
	if !entity.Empty(o.ParanoiaLevel) {
		attrs = append(attrs, entity.Int64(WafProfileParanoiaLevelId, o.ParanoiaLevel))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindWafProfile))
	return
}

func (o *WafProfile) Empty() bool {
	return entity.Empty(o.ParanoiaLevel)
}

func (o *WafProfile) InitSchema(sb *schema.SchemaBuilder) {
	sb.Int64("paranoia_level", "dev.miren.ingress/waf_profile.paranoia_level", schema.Doc("OWASP CRS paranoia level (1-4)"))
}

var (
	KindHttpRoute        = entity.Id("dev.miren.ingress/kind.http_route")
	KindOidcProvider     = entity.Id("dev.miren.ingress/kind.oidc_provider")
	KindPasswordProvider = entity.Id("dev.miren.ingress/kind.password_provider")
	KindWafProfile       = entity.Id("dev.miren.ingress/kind.waf_profile")
	Schema               = entity.Id("dev.miren.ingress/schema.v1alpha")
)

func init() {
	schema.Register("dev.miren.ingress", "v1alpha", func(sb *schema.SchemaBuilder) {
		(&HttpRoute{}).InitSchema(sb)
		(&OidcProvider{}).InitSchema(sb)
		(&PasswordProvider{}).InitSchema(sb)
		(&WafProfile{}).InitSchema(sb)
	})
	schema.RegisterEncodedSchema("dev.miren.ingress", "v1alpha", []byte("\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x94\x96ے\xd30\f\x86_\x04\x86\xe3p&\xcc>Qƍ\x95Dԧ\xb5\xddn{\t3\f\x0f\xc2\xc2\x1b\xc25\x139%\xb1\x9du\xcd͎\xa2H\x9f\xf2ے\xb6\xf7\\1\t\xb7\x1c\x8e\x8dD\v\xaaA5Xp\x0e\xf6\xa8\xb8\xbb?=\xcb\xde|\x9a\xde4\xa3\xf7\xa6\xb5\xfa\xe0\xe1\x17\x11N\x8f\xf2\xc0%&\xd0\xfe\xf4\\K\x86*\xaf\xd6\xf7\b\x82\xbb\xef?v\xc8OOK\xa4\x86\x19C\x05\xbb\xc9\xf0g\x03;\xe4?\xa7\xb4\xf7ŴN0\x94\xaddƠ\x1a\x1c\x97L\x9d\x7f\x13G%o&$vZ\x1a\xad@\xf9Śe\xe6U\x9a\a\xabT\xaa\xfeJ\xaa_\xe5\x9f\x1f\xd3\x02\x9c\xbe\x02\x829}j\xef\xbcE5\x10\xe2\xf5U\xc4\b\x8c\x83%F?\xdb+\xc8p\x04\xebP\xab\xe1xÄ\x19\x990\x16%\xb3\xe7v\xd2!\tu!Q\xbd\x97\xc5\x13\xe7г\x83\xf0Tl\xb8<L\xd5\xf8NkA\x80\x8d\xe6Z\x01F\xedB6'+U\xfb\xae\x98\xac\x91w\xad\xb1\xfa\x88\x17\xc12vͭC\xa8\xa6\x882̹;my\x8c\xbb\xcd\xddk\xe4\x9b\"\xf2\x8e\xf5SZ\x8f\x02\b\xb6_;fL\xf16>/\xb0\xd3\xf3\aFtŜ\x9b\xf7q\x1e\xb9\n\xaal\xd7/\xa4\xefC\x11\xd5\x18f\x99\xd2\xc8Z\x01G\x10a\xd0\x12\xdf$\xb3C\xe5\x8b:\xd7\a\xb3\xd5o$4\xba\xd8Y\xea\x93<6\n\xab\x14\xfb\x8dľ\xbd\x02k:\x81\xa0|\x8b\x9c\x8a\xe3\xf2\x986\xed\xc7J\x92\x83\xceB\xe8~\x19\xbbR\xe2ơ\xc4D\x9a\xa0\xe5O\x9a\xbfq\x91q\xfe\xc5h\x0f6\\\xa4\x88<)oc\x8f\xc5<\xd7i\x03.\xec\xa0ٮ\xdeA\x11ikƨ\x1f\xb2ɜ{\xe2E\x1e\x9f\x85\xfe\xd7\xce\xde\xf8\x80\fx\xed\xfcoj\x18\xff<#sc\xe8\x8a\xd8U{\x82\xf9\xd6J\xc3\xf7n\xd4ַ\xe1\xdf\xffz\xcd\\\xff%\x10\rk\xc5VJ\xae\xb3j\xbcs\x01\xf5m\xf0\x17\x00\x00\xff\xff\x01\x00\x00\xff\xff\xf1\x8a٭\xec\b\x00\x00"))
}
