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
		selfpath: selfpath,
		bathpath: defaultBasePath,
	}
}

func (p *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
