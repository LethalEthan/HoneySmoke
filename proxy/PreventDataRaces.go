package proxy

import "sync/atomic"

func (P *ProxyObject) Close() {
	if P.ServerConn != nil {
		P.ServerConn.Close()
	}
	if P.ClientConn != nil {
		P.ClientConn.Close()
	}
	P.SetClosed()
}

func (P *ProxyObject) GetClosed() bool {
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

func (P *ProxyObject) GetReconnection() bool {
	P.ReconnectionMutex.Lock()
	R := P.Reconnection
	P.ReconnectionMutex.Unlock()
	return R
}

func (P *ProxyObject) SetReconnection(val bool) {
	P.ReconnectionMutex.Lock()
	P.Reconnection = val
	P.ReconnectionMutex.Unlock()
}

func GetLimbo() bool {
	LimboMutex.RLock()
	B := Limbo
	LimboMutex.RUnlock()
	return B
}

func SetLimbo(L bool) {
	LimboMutex.Lock()
	Limbo = L
	LimboMutex.Unlock()
}
