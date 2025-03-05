package mago

type UserContext struct {
	Id       uint           `json:"id"`
	TenantId uint           `json:"tenant_id"`
	Username string         `json:"username"`
	Password string         `json:"password"`
	Mobile   string         `json:"mobile"`
	Ext      map[string]any `json:"ext"`
	Ctx      any
}
