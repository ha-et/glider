package trojan

import (
	"crypto/tls"
	"net"

	"github.com/nadoo/glider/log"
	"github.com/nadoo/glider/pool"
	"github.com/nadoo/glider/proxy"
	"github.com/nadoo/glider/proxy/socks"
)

// NewTrojanDialer returns a trojan proxy dialer.
func NewTrojanDialer(s string, d proxy.Dialer) (proxy.Dialer, error) {
	t, err := NewTrojan(s, d, nil)
	if err != nil {
		log.F("[trojan] create instance error: %s", err)
		return nil, err
	}

	t.tlsConfig = &tls.Config{
		ServerName:         t.serverName,
		InsecureSkipVerify: t.skipVerify,
		NextProtos:         []string{"http/1.1"},
		ClientSessionCache: tls.NewLRUClientSessionCache(64),
		MinVersion:         tls.VersionTLS12,
	}

	return t, err
}

// Addr returns forwarder's address.
func (s *Trojan) Addr() string {
	if s.addr == "" {
		return s.dialer.Addr()
	}
	return s.addr
}

// Dial connects to the address addr on the network net via the proxy.
func (s *Trojan) Dial(network, addr string) (net.Conn, error) {
	return s.dial(network, addr)
}

func (s *Trojan) dial(network, addr string) (net.Conn, error) {
	rc, err := s.dialer.Dial("tcp", s.addr)
	if err != nil {
		log.F("[trojan]: dial to %s error: %s", s.addr, err)
		return nil, err
	}

	tlsConn := tls.Client(rc, s.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}

	buf := pool.GetWriteBuffer()
	defer pool.PutWriteBuffer(buf)

	buf.Write(s.pass[:])
	buf.WriteString("\r\n")

	cmd := socks.CmdConnect
	if network == "udp" {
		cmd = socks.CmdUDPAssociate
	}
	buf.WriteByte(cmd)

	buf.Write(socks.ParseAddr(addr))
	buf.WriteString("\r\n")
	_, err = tlsConn.Write(buf.Bytes())

	return tlsConn, err
}

// DialUDP connects to the given address via the proxy.
func (s *Trojan) DialUDP(network, addr string) (net.PacketConn, net.Addr, error) {
	c, err := s.dial("udp", addr)
	if err != nil {
		return nil, nil, err
	}

	pkc := NewPktConn(c, socks.ParseAddr(addr))
	// TODO: check the addr in return value
	return pkc, nil, nil
}
