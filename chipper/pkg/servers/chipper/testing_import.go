package server

import (

	"net/http/pprof"
	"net/http"
	

)

// TextIntent handles text-based request/responses from the device

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
