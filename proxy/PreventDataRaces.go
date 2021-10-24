package proxy

import "sync/atomic"

func (P *ProxyObject) Close() {
	P.ServerConn.Close()
	P.ClientConn.Close()
	P.SetClosed()
}

func (P *ProxyObject) CheckClosed() bool {
	P.CloseMutex.RLock()
	C := P.Closed
	P.CloseMutex.RUnlock()
	return C
}

func (P *ProxyObject) SetClosed() {
	P.CloseMutex.Lock()
	P.Closed = false
	P.CloseMutex.Unlock()
}

func (P *ProxyObject) GetState() int {
	return int(atomic.LoadUint32(&P.State))
}

func (P *ProxyObject) SetState(S uint32) {
	atomic.StoreUint32(&P.State, S)
}

func (P *ProxyObject) GetCompression() int32 {
	return atomic.LoadInt32(&P.Compression)
}

func (P *ProxyObject) SetCompression(C int32) {
	atomic.StoreInt32(&P.Compression, C)
}
