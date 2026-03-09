package user_v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

var File_delete_request_proto protoreflect.FileDescriptor

var file_delete_request_proto_rawDesc = []byte{
	// name = "delete_request.proto"
	0x0a, 0x14, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x5f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	// package = "user_v1"
	0x12, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x76, 0x31,
	// message DeleteRequest (field 4, len=40)
	0x22, 0x28,
	// name = "DeleteRequest"
	0x0a, 0x0d, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	// field user_id (field 2, len=23)
	0x12, 0x17,
	// name = "user_id"
	0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64,
	// number = 1
	0x18, 0x01,
	// label = LABEL_OPTIONAL
	0x20, 0x01,
	// type = TYPE_STRING
	0x28, 0x09,
	// json_name = "userId"
	0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64,
	// syntax = "proto3"
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_delete_request_proto_rawDescOnce sync.Once
	file_delete_request_proto_rawDescData = file_delete_request_proto_rawDesc
)

func file_delete_request_proto_rawDescGZIP() []byte {
	file_delete_request_proto_rawDescOnce.Do(func() {
		file_delete_request_proto_rawDescData = protoimpl.X.CompressGZIP(file_delete_request_proto_rawDescData)
	})
	return file_delete_request_proto_rawDescData
}

var file_delete_request_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_delete_request_proto_goTypes = []interface{}{
	(*DeleteRequest)(nil),
}
var file_delete_request_proto_depIdxs = []int32{
	0, 0, 0, 0, 0,
}

func init() { file_delete_request_proto_init() }
func file_delete_request_proto_init() {
	if File_delete_request_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_delete_request_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_delete_request_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_delete_request_proto_goTypes,
		DependencyIndexes: file_delete_request_proto_depIdxs,
		MessageInfos:      file_delete_request_proto_msgTypes,
	}.Build()
	File_delete_request_proto = out.File
	file_delete_request_proto_rawDesc = nil
	file_delete_request_proto_goTypes = nil
	file_delete_request_proto_depIdxs = nil
}

type DeleteRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
}

func (x *DeleteRequest) Reset() {
	*x = DeleteRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_delete_request_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteRequest) ProtoMessage() {}

func (x *DeleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_delete_request_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*DeleteRequest) Descriptor() ([]byte, []int) {
	return file_delete_request_proto_rawDescGZIP(), []int{0}
}

func (x *DeleteRequest) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}
