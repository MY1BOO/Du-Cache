package DuCache

import (
	"Du-Cache/DuCache/consistenthash"
	pb "Du-Cache/DuCache/ducachepb"
	"fmt"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultPrefix   = "/ducache/"
	defaultReplicas = 50
)

// 检查 HTTPPool 是否实现了接口 PeerPicker ，若没有则会编译出错
//var _ PeerPicker = (*HttpPool)(nil)

// 检查 httpGetter 是否实现了接口 PeerGetter ，若没有则会编译出错
var _ PeerGetter = (*httpGetter)(nil)

type HttpPool struct {
	//"https://example.net:8000"
	baseURL string
	prefix  string
	mutex   sync.Mutex
	//一致性哈希
	peers *consistenthash.Map
	// key是"http://10.0.0.2:8008"，value是对应的HTTP客户端
	// 即，从一致性哈希里面找到了key存在"http://10.0.0.2:8008"这个远程节点上，利用此字段就可获取到访问这个远程节点的HTTP客户端
	httpGetters map[string]*httpGetter
}

func NewHttpPool(baseURL string) *HttpPool {
	return &HttpPool{
		// 记录自己的地址，包括主机名/IP和端口
		baseURL: baseURL,
		// 作为节点间通讯地址的前缀
		prefix: defaultPrefix,
	}
}

func (p *HttpPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.baseURL, fmt.Sprintf(format, v...))
}

/*
约定访问路径格式为/<basepath>/<groupname>/<key>
通过groupname得到group实例，再使用group.Get(key)获取缓存数据
*/
func (p *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 判断访问路径的前缀是否是prefix，不是则返回错误信息
	if !strings.HasPrefix(r.URL.Path, p.prefix) {
		panic("HttpPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	strs := strings.SplitN(r.URL.Path[len(p.prefix):], "/", 2)
	if len(strs) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := strs[0]
	key := strs[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	byteView, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := proto.Marshal(&pb.Response{Value: byteView.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

//实例化一致性哈希map，将节点添加进去，并将节点和对应的httpGetter映射起来
// peers是一个字符串切片["http://localhost:8001","http://localhost:8002","http://localhost:8003"]
func (p *HttpPool) Set(peers ...string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter)
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.prefix}
	}

}

func (p *HttpPool) PickPeer(key string) (PeerGetter, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	//一致性哈希获取节点URL
	peer := p.peers.Get(key)
	//如果不为空和不为当前节点，返回对应的httpGetter
	if peer != "" && peer != p.baseURL {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

//HTTP客户端，实现了PeerGetter接口
type httpGetter struct {
	//http://example.com/ducache
	baseURL string
}

func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	//进行格式化字符串拼接，格式：http://example.com/ducache/group/key
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.Group),
		url.QueryEscape(in.Key),
	)
	//发起http请求
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	//检查状态码
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}
	//读取相应消息体内容
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	//反序列化
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}
