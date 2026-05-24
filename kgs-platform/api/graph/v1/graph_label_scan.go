package v1

// GetNodesByLabelRequest is the request message for GetNodesByLabel RPC.
// Hand-written until protoc regeneration.
type GetNodesByLabelRequest struct {
	Label     string `protobuf:"bytes,1,opt,name=label,proto3" json:"label,omitempty"`
	PageSize  int32  `protobuf:"varint,2,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	PageToken string `protobuf:"bytes,3,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
}

func (x *GetNodesByLabelRequest) GetLabel() string {
	if x != nil {
		return x.Label
	}
	return ""
}

func (x *GetNodesByLabelRequest) GetPageSize() int32 {
	if x != nil {
		return x.PageSize
	}
	return 0
}

func (x *GetNodesByLabelRequest) GetPageToken() string {
	if x != nil {
		return x.PageToken
	}
	return ""
}

// GetNodesByLabelReply is the response message for GetNodesByLabel RPC.
type GetNodesByLabelReply struct {
	Nodes         []*GraphNode `protobuf:"bytes,1,rep,name=nodes,proto3" json:"nodes,omitempty"`
	Total         int32        `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
	NextPageToken string       `protobuf:"bytes,3,opt,name=next_page_token,json=nextPageToken,proto3" json:"next_page_token,omitempty"`
}

func (x *GetNodesByLabelReply) GetNodes() []*GraphNode {
	if x != nil {
		return x.Nodes
	}
	return nil
}

func (x *GetNodesByLabelReply) GetTotal() int32 {
	if x != nil {
		return x.Total
	}
	return 0
}

func (x *GetNodesByLabelReply) GetNextPageToken() string {
	if x != nil {
		return x.NextPageToken
	}
	return ""
}

// UpdateNodeRequest is the request message for UpdateNode RPC.
type UpdateNodeRequest struct {
	NodeId         string `protobuf:"bytes,1,opt,name=node_id,json=nodeId,proto3" json:"node_id,omitempty"`
	PropertiesJson string `protobuf:"bytes,2,opt,name=properties_json,json=propertiesJson,proto3" json:"properties_json,omitempty"`
	Label          string `protobuf:"bytes,3,opt,name=label,proto3" json:"label,omitempty"`
}

func (x *UpdateNodeRequest) GetNodeId() string {
	if x != nil {
		return x.NodeId
	}
	return ""
}

func (x *UpdateNodeRequest) GetPropertiesJson() string {
	if x != nil {
		return x.PropertiesJson
	}
	return ""
}

func (x *UpdateNodeRequest) GetLabel() string {
	if x != nil {
		return x.Label
	}
	return ""
}

// UpdateNodeReply is the response message for UpdateNode RPC.
type UpdateNodeReply struct {
	NodeId         string `protobuf:"bytes,1,opt,name=node_id,json=nodeId,proto3" json:"node_id,omitempty"`
	Label          string `protobuf:"bytes,2,opt,name=label,proto3" json:"label,omitempty"`
	PropertiesJson string `protobuf:"bytes,3,opt,name=properties_json,json=propertiesJson,proto3" json:"properties_json,omitempty"`
}

func (x *UpdateNodeReply) GetNodeId() string {
	if x != nil {
		return x.NodeId
	}
	return ""
}

func (x *UpdateNodeReply) GetLabel() string {
	if x != nil {
		return x.Label
	}
	return ""
}

func (x *UpdateNodeReply) GetPropertiesJson() string {
	if x != nil {
		return x.PropertiesJson
	}
	return ""
}
