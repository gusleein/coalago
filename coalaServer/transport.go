package coalaServer

import (
	"net"
	"time"

	cerr "github.com/gusleein/coalago/errors"
	"github.com/gusleein/coalago/util"
)

func (s *Server) sendPacketsToAddr(pc net.PacketConn, packets []*packet, windowsize int, shift int, addr net.Addr) error {
	stop := shift + windowsize
	if stop >= len(packets) {
		stop = len(packets)
	}

	if shift == len(packets) {
		return cerr.MaxAttempts
	}

	for i := 0; i < stop; i++ {
		if packets[i].acked {
			continue
		}

		if time.Since(packets[i].lastSend) < timeWait {
			continue
		}

		if packets[i].attempts == maxSendAttempts {
			util.MetricExpiredMessages.Inc()
			return cerr.MaxAttempts
		}
		packets[i].attempts++
		if packets[i].attempts > 1 {
			util.MetricRetransmitMessages.Inc()
		}
		packets[i].lastSend = time.Now()
		m := packets[i].message

		if err := s.send(pc, m, addr); err != nil {
			return err
		}

	}

	return nil
}
