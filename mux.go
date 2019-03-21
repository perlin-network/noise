package noise

import (
	"crypto/rand"
	"runtime"
	"sync/atomic"
	"time"
)

type PeerMux struct {
	peer *Peer
	cid  ChannelID
}

func NewPeerMux(p *Peer) *PeerMux {
	var cid ChannelID
	_, err := rand.Read(cid[:])
	if err != nil {
		panic(err)
	}

	p.peerMuxAlive.Store(cid, struct{}{})
	m := &PeerMux{
		peer: p,
		cid:  cid,
	}
	m.gcSelfRegister()
	return m
}

func (m *PeerMux) gcSelfRegister() {
	runtime.SetFinalizer(m, func(m *PeerMux) {
		m.peer.peerMuxAlive.Delete(m.cid)
	})
}

func (m *PeerMux) SendMessage(message Message) error {
	return m.peer.SendMessageMux(m.cid, message)
}

func (m *PeerMux) Receive(o Opcode) <-chan Message {
	return m.peer.ReceiveMux(m.cid, o)
}

func ListenForPeerMux(p *Peer, o Opcode, cb func(m *PeerMux)) {
	for {
		select {
		case cid := <-p.WaitForMuxChannel(o):
			_, loaded := p.peerMuxAlive.LoadOrStore(cid, struct{}{})
			if !loaded {
				m := &PeerMux{
					peer: p,
					cid:  cid,
				}
				m.gcSelfRegister()
				cb(m)
			}
		case <-time.After(5 * time.Second):
			if atomic.LoadUint32(&p.killOnce) != 0 {
				break
			}
		}
	}
}
