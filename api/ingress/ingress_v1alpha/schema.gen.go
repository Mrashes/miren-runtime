package ingress_v1alpha

import (
	entity "miren.dev/runtime/pkg/entity"
	schema "miren.dev/runtime/pkg/entity/schema"
)

const (
	HttpRouteAppId           = entity.Id("dev.miren.ingress/http_route.app")
	HttpRouteAuthProviderId  = entity.Id("dev.miren.ingress/http_route.auth_provider")
	HttpRouteClaimMappingsId = entity.Id("dev.miren.ingress/http_route.claim_mappings")
	HttpRouteDefaultId       = entity.Id("dev.miren.ingress/http_route.default")
	HttpRouteHostId          = entity.Id("dev.miren.ingress/http_route.host")
	HttpRouteWafProfileId    = entity.Id("dev.miren.ingress/http_route.waf_profile")
)

type HttpRoute struct {
	ID            entity.Id       `json:"id"`
	App           entity.Id       `cbor:"app,omitempty" json:"app,omitempty"`
	AuthProvider  entity.Id       `cbor:"auth_provider,omitempty" json:"auth_provider,omitempty"`
	ClaimMappings []ClaimMappings `cbor:"claim_mappings,omitempty" json:"claim_mappings,omitempty"`
	Default       bool            `cbor:"default,omitempty" json:"default,omitempty"`
	Host          string          `cbor:"host,omitempty" json:"host,omitempty"`
	WafProfile    entity.Id       `cbor:"waf_profile,omitempty" json:"waf_profile,omitempty"`
}

func (o *HttpRoute) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(HttpRouteAppId); ok && a.Value.Kind() == entity.KindId {
		o.App = a.Value.Id()
	}
	if a, ok := e.Get(HttpRouteAuthProviderId); ok && a.Value.Kind() == entity.KindId {
		o.AuthProvider = a.Value.Id()
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
	if !entity.Empty(o.AuthProvider) {
		attrs = append(attrs, entity.Ref(HttpRouteAuthProviderId, o.AuthProvider))
	}
	for _, v := range o.ClaimMappings {
		attrs = append(attrs, entity.Component(HttpRouteClaimMappingsId, v.Encode()))
	}
	attrs = append(attrs, entity.Bool(HttpRouteDefaultId, o.Default))
	if !entity.Empty(o.Host) {
		attrs = append(attrs, entity.String(HttpRouteHostId, o.Host))
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
	if !entity.Empty(o.AuthProvider) {
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
	if !entity.Empty(o.WafProfile) {
		return false
	}
	return true
}

func (o *HttpRoute) InitSchema(sb *schema.SchemaBuilder) {
	sb.Ref("app", "dev.miren.ingress/http_route.app", schema.Doc("The application to route to"), schema.Indexed, schema.Tags("dev.miren.app_ref"))
	sb.Ref("auth_provider", "dev.miren.ingress/http_route.auth_provider", schema.Doc("Reference to an auth provider (OIDC or password) for authentication"), schema.Indexed)
	sb.Component("claim_mappings", "dev.miren.ingress/http_route.claim_mappings", schema.Doc("Mappings from JWT claims to HTTP headers"), schema.Many)
	(&ClaimMappings{}).InitSchema(sb.Builder("http_route.claim_mappings"))
	sb.Bool("default", "dev.miren.ingress/http_route.default", schema.Doc("Whether this is the default route for routing"), schema.Indexed)
	sb.String("host", "dev.miren.ingress/http_route.host", schema.Doc("The hostname to match on for the application"), schema.Indexed)
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
	schema.RegisterEncodedSchema("dev.miren.ingress", "v1alpha", []byte("\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x94\x95ے\xd30\f\x86_\x04\x86\xe3p&\xcc>Qƍ\x95Dԧ\xb5\xddl{\t3\xf0\",\xbc!\\3\x91\xd3Ml\xa7\xa9\xf7\xa6\xa3\xca\xf2'K\xfe\xe5\xdcs\xc5$\xdcr\x18*\x89\x16T\x85\xaa\xb3\xe0\x1c\xecQqw\x7f|\x91\xad|\x19W\xaa\xde{S[}\xf0\xf0\x87\b\xc7'y\xe0\x1c\x13h\xffZ\xae%C\x95gk[\x04\xc1\xdd\xcf_;\xe4\xc7\xe7[\xa4\x8a\x19C\t\x9b\xd1\xf0'\x03;\xe4\xb4\xed\xc3\xf6\xb6\x83\xefkc\xf5\x80\x1c,\x01d\xec\x9aP\xbfG\xd4\xc7MT#\x18\xcaZ2cPu\x8eK\xa6N\x7f\x89\xa8\x92\x95\x11\x89\x8d\x96F+P~\xb6\xa6\x8e\xe5Y\xaa\x8bY\n\x1b\xf8\x9d:\xf1&?~L\vp:\x05\x04s<j\xeb\xbcE\xd5\x11\xe2\xedUD\x0f\xec\xdc\xc9v\xb2\x17\x90n\x00\xebP\xabn\xb8a\xc2\xf4L\x18\x8b\x92\xd9S=\xd6!\tu&Q\xbeכ\x1d\xe7в\x83\xf0\x94\xac;\xff\x19\xb3\xf1\x9dւ\x00+:]\x00z\xed\xc2nNVZ\xed\xbb\xcd\xcdw\xac\x1deҢ\x00b엎I6\x9b\xf5~\x9daǗ\x17\xe6i\xc1\x9c\xe4\xf14\x8f\\\x04\x15\n\xe2\x1b\xd5\xf7i\x13U\x19f\x99\xd2\xc8j\x01\x03\x88 \xe5\xc47\x96٠\xf2\x9bu.\x1b\xb3v\xa3T\xa8F\xde<L\xddT\xea\xb3<6\n+,\xf6\a\x15\xfb\xfe\n\xacj\x04\x82\xf25rJ\x8e\xf3\xdfT\x16\x9f\vI\x0e\x1a\vA_2v\xa5ĕ\xa6\xc4D\xd2\xe8\xfc\x93\xee_\xb9\xc8x\xff٨\x0f6\\\xa4\x88<)o奈y\xae\xd1\x06\\\x98\xf2\xc9.\x9e\xf2\x88\xb46c\xa4\aÜ\xbbӖ\xa7\x9ax\x95\xc7g\xa1\x8fz\x15W\x0e\x90\x01\xaf\xf5\xff\xa6\x84\xf1\xe0\xe9\x99\xeb\x83*bWi\ao3v\x1a\xbew\xbd\xb6\xbe\x0e\xdf\xea\xe53s\xfd\xb3\x1d\rk\xc1\xab\x94\\g\xd1x\xe7\x05\x94\xcb\xe0?\x00\x00\x00\xff\xff\x01\x00\x00\xff\xff;\xd8hV\x99\b\x00\x00"))
}
