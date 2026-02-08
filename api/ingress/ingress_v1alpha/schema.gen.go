package ingress_v1alpha

import (
	entity "miren.dev/runtime/pkg/entity"
	schema "miren.dev/runtime/pkg/entity/schema"
)

const (
	HttpRouteAppId        = entity.Id("dev.miren.ingress/http_route.app")
	HttpRouteDefaultId    = entity.Id("dev.miren.ingress/http_route.default")
	HttpRouteHostId       = entity.Id("dev.miren.ingress/http_route.host")
	HttpRouteOidcConfigId = entity.Id("dev.miren.ingress/http_route.oidc_config")
)

type HttpRoute struct {
	ID         entity.Id  `json:"id"`
	App        entity.Id  `cbor:"app,omitempty" json:"app,omitempty"`
	Default    bool       `cbor:"default,omitempty" json:"default,omitempty"`
	Host       string     `cbor:"host,omitempty" json:"host,omitempty"`
	OidcConfig OidcConfig `cbor:"oidc_config,omitempty" json:"oidc_config,omitempty"`
}

func (o *HttpRoute) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(HttpRouteAppId); ok && a.Value.Kind() == entity.KindId {
		o.App = a.Value.Id()
	}
	if a, ok := e.Get(HttpRouteDefaultId); ok && a.Value.Kind() == entity.KindBool {
		o.Default = a.Value.Bool()
	}
	if a, ok := e.Get(HttpRouteHostId); ok && a.Value.Kind() == entity.KindString {
		o.Host = a.Value.String()
	}
	if a, ok := e.Get(HttpRouteOidcConfigId); ok && a.Value.Kind() == entity.KindComponent {
		o.OidcConfig.Decode(a.Value.Component())
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
	attrs = append(attrs, entity.Bool(HttpRouteDefaultId, o.Default))
	if !entity.Empty(o.Host) {
		attrs = append(attrs, entity.String(HttpRouteHostId, o.Host))
	}
	if !o.OidcConfig.Empty() {
		attrs = append(attrs, entity.Component(HttpRouteOidcConfigId, o.OidcConfig.Encode()))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindHttpRoute))
	return
}

func (o *HttpRoute) Empty() bool {
	if !entity.Empty(o.App) {
		return false
	}
	if !entity.Empty(o.Default) {
		return false
	}
	if !entity.Empty(o.Host) {
		return false
	}
	if !o.OidcConfig.Empty() {
		return false
	}
	return true
}

func (o *HttpRoute) InitSchema(sb *schema.SchemaBuilder) {
	sb.Ref("app", "dev.miren.ingress/http_route.app", schema.Doc("The application to route to"), schema.Indexed, schema.Tags("dev.miren.app_ref"))
	sb.Bool("default", "dev.miren.ingress/http_route.default", schema.Doc("Whether this is the default route for routing"), schema.Indexed)
	sb.String("host", "dev.miren.ingress/http_route.host", schema.Doc("The hostname to match on for the application"), schema.Indexed)
	sb.Component("oidc_config", "dev.miren.ingress/http_route.oidc_config", schema.Doc("OIDC authentication configuration for this route"))
	(&OidcConfig{}).InitSchema(sb.Builder("http_route.oidc_config"))
}

const (
	OidcConfigClaimMappingsId = entity.Id("dev.miren.ingress/oidc_config.claim_mappings")
	OidcConfigClientIdId      = entity.Id("dev.miren.ingress/oidc_config.client_id")
	OidcConfigClientSecretId  = entity.Id("dev.miren.ingress/oidc_config.client_secret")
	OidcConfigProviderUrlId   = entity.Id("dev.miren.ingress/oidc_config.provider_url")
	OidcConfigScopesId        = entity.Id("dev.miren.ingress/oidc_config.scopes")
)

type OidcConfig struct {
	ClaimMappings []ClaimMappings `cbor:"claim_mappings,omitempty" json:"claim_mappings,omitempty"`
	ClientId      string          `cbor:"client_id,omitempty" json:"client_id,omitempty"`
	ClientSecret  string          `cbor:"client_secret,omitempty" json:"client_secret,omitempty"`
	ProviderUrl   string          `cbor:"provider_url,omitempty" json:"provider_url,omitempty"`
	Scopes        string          `cbor:"scopes,omitempty" json:"scopes,omitempty"`
}

func (o *OidcConfig) Decode(e entity.AttrGetter) {
	for _, a := range e.GetAll(OidcConfigClaimMappingsId) {
		if a.Value.Kind() == entity.KindComponent {
			var v ClaimMappings
			v.Decode(a.Value.Component())
			o.ClaimMappings = append(o.ClaimMappings, v)
		}
	}
	if a, ok := e.Get(OidcConfigClientIdId); ok && a.Value.Kind() == entity.KindString {
		o.ClientId = a.Value.String()
	}
	if a, ok := e.Get(OidcConfigClientSecretId); ok && a.Value.Kind() == entity.KindString {
		o.ClientSecret = a.Value.String()
	}
	if a, ok := e.Get(OidcConfigProviderUrlId); ok && a.Value.Kind() == entity.KindString {
		o.ProviderUrl = a.Value.String()
	}
	if a, ok := e.Get(OidcConfigScopesId); ok && a.Value.Kind() == entity.KindString {
		o.Scopes = a.Value.String()
	}
}

func (o *OidcConfig) Encode() (attrs []entity.Attr) {
	for _, v := range o.ClaimMappings {
		attrs = append(attrs, entity.Component(OidcConfigClaimMappingsId, v.Encode()))
	}
	if !entity.Empty(o.ClientId) {
		attrs = append(attrs, entity.String(OidcConfigClientIdId, o.ClientId))
	}
	if !entity.Empty(o.ClientSecret) {
		attrs = append(attrs, entity.String(OidcConfigClientSecretId, o.ClientSecret))
	}
	if !entity.Empty(o.ProviderUrl) {
		attrs = append(attrs, entity.String(OidcConfigProviderUrlId, o.ProviderUrl))
	}
	if !entity.Empty(o.Scopes) {
		attrs = append(attrs, entity.String(OidcConfigScopesId, o.Scopes))
	}
	return
}

func (o *OidcConfig) Empty() bool {
	if len(o.ClaimMappings) != 0 {
		return false
	}
	if !entity.Empty(o.ClientId) {
		return false
	}
	if !entity.Empty(o.ClientSecret) {
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

func (o *OidcConfig) InitSchema(sb *schema.SchemaBuilder) {
	sb.Component("claim_mappings", "dev.miren.ingress/oidc_config.claim_mappings", schema.Doc("Mappings from JWT claims to HTTP headers"), schema.Many)
	(&ClaimMappings{}).InitSchema(sb.Builder("oidc_config.claim_mappings"))
	sb.String("client_id", "dev.miren.ingress/oidc_config.client_id", schema.Doc("The OAuth2 client ID"))
	sb.String("client_secret", "dev.miren.ingress/oidc_config.client_secret", schema.Doc("The OAuth2 client secret"))
	sb.String("provider_url", "dev.miren.ingress/oidc_config.provider_url", schema.Doc("The OIDC provider URL (e.g. https://accounts.google.com)"))
	sb.String("scopes", "dev.miren.ingress/oidc_config.scopes", schema.Doc("Space-separated list of OAuth2 scopes (e.g. \"openid email profile\")"))
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

var (
	KindHttpRoute = entity.Id("dev.miren.ingress/kind.http_route")
	Schema        = entity.Id("dev.miren.ingress/schema.v1alpha")
)

func init() {
	schema.Register("dev.miren.ingress", "v1alpha", func(sb *schema.SchemaBuilder) {
		(&HttpRoute{}).InitSchema(sb)
	})
	schema.RegisterEncodedSchema("dev.miren.ingress", "v1alpha", []byte("\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x8c\x94\xdfN\xeb0\f\xc6_\xe4\\\x9c#\x1d\x01\x02T\xc4\x15\x8fSe\xb5ۚ\xe5\x1fIVm\xb7\b\x1e\x041xC\xb8Fq65mG\xd7;7\xf1\xf7\xf3g'\xcd\x1e\xb4P\xf8\x04\xd8\x15\x8a\x1c\xea\x82t\xe3\xd0{\\\x93\x06\xff\xb6\xfd7ٹ\x8b;E\x1b\x82-\x9d\xd9\x04\xfcd\xc2\xf6\xcf4\xb1\xcfI\xb4\xef\x1a\x8c\x12\xa4\xa7\xd5\xea\x9aP\x82\x7fy_\x11l\xffΑ\na-\x17\xacb\x10v\x16W\x04,\xfb?+\x03\xac\xc5F\x06\x966Ǐ(\x87\x951\x92\x01'Z\xcd\x00\xad\xf1I\r\x1cEi\xed\x83#\xdd\xec\xa3\xf8jVl\b\xaa\xb22\xba\xa6\x86\x19\xeb|!\xa2\xa82\xca\x1a\x8d:\xf4\xd1a\xb0Srq\x9a\xbcpȯ\x1f\xd1\xef\xed\xd4o\x86**)H\x95JXK\xba\xf1\xa0\x84\xde}\xb1\x1d=\xda9c\xfea\xa9\xf9QŅ\xbd<\xf3\xc1]L{\x19\xd2\x12\x9c\x1da\n\xb3\xe3c\xc4\xe5YD\x8b\x02\xd01\xa3>\xc4\x19\xa4\xe9\xd0y2\xba\xe9\ue174\xad\x90֑\x12nW\xc6>\x14\xa3\x8e\xa4\xdf\xea\r\x87A\xa8CI\xc0\xf5\xa8\xff\x1c\xfb\xbeY\xc4\xf1X9L\xb7W\r\x97Ƽ\xeby\x9eu\xa6#@Wn\x9cd\x9c\x1c\xac\x8ci'~ɜ\xe6+cѧ\x91\x1e\xe2\xa5#}\x8c\x9c\x84\x99\xcf\xebo\xdb8o\xed[\xe3B\x99\u07b9,o\xc1\x93\xf7\x03\x00\x00\xff\xff\x01\x00\x00\xff\xff\x15e\xa3%5\x05\x00\x00"))
}
