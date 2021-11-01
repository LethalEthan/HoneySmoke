package proxy

import (
	"net"
	"sync/atomic"
)

func (P *ProxyObject) Close() {
	P.PacketPerSecondC <- 1 //closes pps routine
	if P.ServerConn != nil {
		P.ServerConn.Close()
	}
	if P.ClientConn != nil {
		MainProxy.Delete(P.ClientConn.RemoteAddr().String())
		P.ClientConn.Close()
	}
	P.SetClosed()
}

func (P *ProxyObject) GetClosed() bool {
	P.CloseMutex.RLock()
	C := !P.Closed
	P.CloseMutex.RUnlock()
	return C
}

func (P *ProxyObject) SetClosed() {
	P.CloseMutex.Lock()
	P.Closed = true
	P.ClientConn = nil
	P.ServerConn = nil
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

func (P *Proxy) GetLimbo() bool {
	P.LimboMutex.RLock()
	B := P.Limbo
	P.LimboMutex.RUnlock()
	return B
}

func (P *Proxy) SetLimbo(L bool) {
	P.LimboMutex.Lock()
	P.Limbo = L
	P.LimboMutex.Unlock()
}

func (P *Proxy) GetListener() net.Listener {
	P.ListenerMutex.Lock()
	L := P.Listener
	P.ListenerMutex.Unlock()
	return L
}

func (P *Proxy) SetListener(val net.Listener) {
	P.ListenerMutex.Lock()
	P.Listener = val
	P.ListenerMutex.Unlock()
}

func (P *Proxy) Delete(key string) {
	if _, i := P.ProxyObjects[key]; i {
		delete(P.ProxyObjects, key)
	}
}

func (P *Proxy) Set(key string, val ProxyObject) {
	P.ProxyObjectsMutex.Lock()
	P.ProxyObjects[key] = val
	P.ProxyObjectsMutex.Unlock()
}

func (P *Proxy) Get(key string) ProxyObject {
	P.ProxyObjectsMutex.RLock()
	PO := P.ProxyObjects[key]
	P.ProxyObjectsMutex.RUnlock()
	return PO
}
