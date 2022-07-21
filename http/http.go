package http

import (
	"Du-Cache/group"
	"net/http"
	"strings"
)

const defaultBasePath = "/ducache/"

type HttpPool struct {
	selfpath string
	bathpath string
}

func NewHttpPool(selfpath string) *HttpPool {
	return &HttpPool{
		// 记录自己的地址，包括主机名/IP和端口
		selfpath: selfpath,
		// 作为节点间通讯地址的前缀
		bathpath: defaultBasePath,
	}
}

/*
约定访问路径格式为/<basepath>/<groupname>/<key>
通过groupname得到group实例，再使用group.Get(key)获取缓存数据
*/
func (p *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 判断访问路径的前缀是否是basePath，不是则返回错误信息
	if !strings.HasPrefix(r.URL.Path, p.bathpath) {
		panic("HttpPool serving unexpected path: " + r.URL.Path)
	}
	strs := strings.SplitN(r.URL.Path[len(p.bathpath):], "/", 2)
	if len(strs) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := strs[0]
	key := strs[1]

	group := group.GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	byteView, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(byteView.ByteSlice())
}
