package ingress_v1alpha

import (
	entity "miren.dev/runtime/pkg/entity"
	schema "miren.dev/runtime/pkg/entity/schema"
)

const (
	HttpRouteAppId           = entity.Id("dev.miren.ingress/http_route.app")
	HttpRouteClaimMappingsId = entity.Id("dev.miren.ingress/http_route.claim_mappings")
	HttpRouteDefaultId       = entity.Id("dev.miren.ingress/http_route.default")
	HttpRouteHostId          = entity.Id("dev.miren.ingress/http_route.host")
	HttpRouteOidcProviderId  = entity.Id("dev.miren.ingress/http_route.oidc_provider")
	HttpRouteWafLevelId      = entity.Id("dev.miren.ingress/http_route.waf_level")
)

type HttpRoute struct {
	ID            entity.Id       `json:"id"`
	App           entity.Id       `cbor:"app,omitempty" json:"app,omitempty"`
	ClaimMappings []ClaimMappings `cbor:"claim_mappings,omitempty" json:"claim_mappings,omitempty"`
	Default       bool            `cbor:"default,omitempty" json:"default,omitempty"`
	Host          string          `cbor:"host,omitempty" json:"host,omitempty"`
	OidcProvider  entity.Id       `cbor:"oidc_provider,omitempty" json:"oidc_provider,omitempty"`
	WafLevel      int64           `cbor:"waf_level,omitempty" json:"waf_level,omitempty"`
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
	if a, ok := e.Get(HttpRouteWafLevelId); ok && a.Value.Kind() == entity.KindInt64 {
		o.WafLevel = a.Value.Int64()
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
	if !entity.Empty(o.WafLevel) {
		attrs = append(attrs, entity.Int64(HttpRouteWafLevelId, o.WafLevel))
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
	if !entity.Empty(o.WafLevel) {
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
	sb.Int64("waf_level", "dev.miren.ingress/http_route.waf_level", schema.Doc("WAF protection level (0=disabled, 1-4=OWASP CRS paranoia level)"))
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

var (
	KindHttpRoute    = entity.Id("dev.miren.ingress/kind.http_route")
	KindOidcProvider = entity.Id("dev.miren.ingress/kind.oidc_provider")
	Schema           = entity.Id("dev.miren.ingress/schema.v1alpha")
)

func init() {
	schema.Register("dev.miren.ingress", "v1alpha", func(sb *schema.SchemaBuilder) {
		(&HttpRoute{}).InitSchema(sb)
		(&OidcProvider{}).InitSchema(sb)
	})
	schema.RegisterEncodedSchema("dev.miren.ingress", "v1alpha", []byte("\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x8c\x94뮔0\x10\xc7_\xc4DM\x8c\xc6\x1b\xc6'j\xba\xed\x00\xe3\xf6f\xdb\xc5ݯ&\xfa ^\xce\x1b\x9e\xf3\xf9\xa4S\b\x14X\xe0\v\x99\xb63\xbf\xb9\xf0o\xffI\xc35|\x97\xd0U\x1a=\x98\nM\xe3!\x048\xa3\x91\xe1\xcf\xf5\xf5\xe2\xe4K:\xa9\xda\x18\x1d\xf3\xf6\x12\xe1\x81\b\xd7\x17K\xc7\xd1'Ӟji5G\xb3\xccV\xd7\bJ\x86\xdf\x7fO(\xaf\xaf\xb6H\x15w\x8e\x12\x8adě\x83\x13\xca\xff)\xec\xe3f\x98P\x1c5\xd3\xdc94M\x90\x9a\x9b\xdb#q\xcc\xec$!QX\xed\xac\x01\x13G\xabos\x99\xa5\xba\x9b\xe5`\xd7?\xa9\xeb\xb7\xcb\xf2KZ\x86S\x15\x90\xcdTj\x1d\xa2G\xd3\x10\xe2\xdd.\xa2\x05.\xc1\x13\xa3\xee\xed\t\xa4\xe9\xc0\a\xb4\xa6\xe9\xber\xe5Z\xae\x9cG\xcd\xfd\x8d\xa5>4\xa1\x06\x12\xe5{\xb39q\t5\xbf\xa8Hɚa\x91\xb2ɓ\xb5\x8a\x00+\xe2\x9a\x00Z\x1br\xb4$k\xde\xed\x87\xcd`\x8bR0\xe7m\x87Cú\xdc\xea\xa5so\xf6\x13\xd4\x0f^3\x05\x1d(\xc2\xe0\xb8L\b\x81&n\xce\xed\xdbHZ\x1b\x18]\xa6\xa2\xb2^h/\x97\xbe\x85\xdbAq\xfd\xa2\x06\xdf\xef\xc0*\xa1\x10Ld(s\x8f\xe3r>\xf5\xcf\aI\x01\x84\x87\xfc\xfbt\xb95'\xae\f\xa5$\x92\x04\xc6\xcf<\xfe\xd3^\xfc`\xb0\x8bϿP\x15;sފ\x18J^\x10\xd6Aȗ\xa8\xb7\x0f_\xa2\x824w=\x87\xd6\xfa\xc8\xf2\xcb;\xd5\xcd\xfe#\\\x82\x8f\t\xed\x19\x00\x00\xff\xff\x01\x00\x00\xff\xff\x87\xbd)}\xfb\x05\x00\x00"))
}
