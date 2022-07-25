package DuCache

import pb "Du-Cache/DuCache/ducachepb"

// PeerPicker 是必须实现的接口，用于查找拥有特定key的节点
// 此项目中由 HttpPool 实现此接口，HttpPool是HTTP池（也可理解为HTTP服务器）
type PeerPicker interface {
	// PickPeer 根据传入的key选择相应节点PeerGetter（http客户端）
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// http客户端接口，http客户端必须实现Get方法
// 此项目中的http客户端类是httpGetter
type PeerGetter interface {
	// Get 从对应group查找缓存值
	Get(in *pb.Request, out *pb.Response) error
}
