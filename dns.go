package main

import (
	"github.com/miekg/dns"
	"net"
)

func localhostDNSHandler(w dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(req)

	m.Answer = make([]dns.RR, 1)
	ip := net.IPv4(127, 0, 0, 1)
	m.Answer[0] = &dns.A{Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}, A: ip}

	w.WriteMsg(m)
}

func ListenAndServeDNS(address string) error {
	dns.HandleFunc("dev.", localhostDNSHandler)

	err := dns.ListenAndServe(address, "udp", nil)
	if err != nil {
		return err
	}
	return nil
}
