package user

import "github.com/maliboot/mago"

type MemberTokenRequest struct {
	mago.AbstractDTO
	Email          string `json:"email"`
	Password       string `json:"password"`
	Ext            string `json:"ext"`
	IsExtendExpire bool
}

type MemberTokenResponse struct {
	mago.AbstractViewObject
	Token  string `json:"token"`
	Expire string `json:"expire"`
}

type MemberResponse struct {
	mago.AbstractViewObject
	Email string `json:"email"`
}
