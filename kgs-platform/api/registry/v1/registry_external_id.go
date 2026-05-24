package v1

// GetAppByExternalIDRequest is the request message for GetAppByExternalID.
type GetAppByExternalIDRequest struct {
	ExternalId string `protobuf:"bytes,1,opt,name=external_id,json=externalId,proto3" json:"external_id,omitempty"`
}

func (x *GetAppByExternalIDRequest) GetExternalId() string {
	if x != nil {
		return x.ExternalId
	}
	return ""
}
