/*
   rtldavis, an rtl-sdr receiver for Davis Instruments weather stations.
   Copyright (C) 2015  Douglas Hall

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
package protocol

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/bemasher/rtlamr/crc"
	"github.com/bemasher/rtldavis/dsp"
)

func NewPacketConfig(symbolLength int) (cfg dsp.PacketConfig) {
	return dsp.NewPacketConfig(
		19200,
		14,
		16,
		80,
		"1100101110001001",
	)
}

type Parser struct {
	dsp.Demodulator
	crc.CRC

	Cfg dsp.PacketConfig

	ID        int
	DwellTime time.Duration

	channelCount int
	channels     []int

	hopIdx     int
	hopPattern []int

	currentPPM int
	channelPPM map[int]int
}

func NewParser(symbolLength, id int) (p Parser) {
	p.Cfg = NewPacketConfig(symbolLength)
	p.Demodulator = dsp.NewDemodulator(&p.Cfg)
	p.CRC = crc.NewCRC("CCITT-16", 0, 0x1021, 0)

	p.channels = []int{
		902355835, 902857585, 903359336, 903861086, 904362837, 904864587,
		905366338, 905868088, 906369839, 906871589, 907373340, 907875090,
		908376841, 908878591, 909380342, 909882092, 910383843, 910885593,
		911387344, 911889094, 912390845, 912892595, 913394346, 913896096,
		914397847, 914899597, 915401347, 915903098, 916404848, 916906599,
		917408349, 917910100, 918411850, 918913601, 919415351, 919917102,
		920418852, 920920603, 921422353, 921924104, 922425854, 922927605,
		923429355, 923931106, 924432856, 924934607, 925436357, 925938108,
		926439858, 926941609, 927443359,
	}
	p.channelCount = len(p.channels)

	p.hopIdx = rand.Intn(p.channelCount)
	p.hopPattern = []int{
		0, 19, 41, 25, 8, 47, 32, 13, 36, 22, 3, 29, 44, 16, 5, 27, 38, 10,
		49, 21, 2, 30, 42, 14, 48, 7, 24, 34, 45, 1, 17, 39, 26, 9, 31, 50,
		37, 12, 20, 33, 4, 43, 28, 15, 35, 6, 40, 11, 23, 46, 18,
	}

	p.channelPPM = make(map[int]int)

	p.ID = id
	p.DwellTime = 2562500 * time.Microsecond
	p.DwellTime += time.Duration(p.ID) * 62500 * time.Microsecond

	return
}

func (p *Parser) channelFreq(hopIdx int) int {
	return p.channels[p.hopPattern[hopIdx]]
}

func (p *Parser) NextChannel() int {
	p.hopIdx = (p.hopIdx + 1) % p.channelCount
	return p.channelFreq(p.hopIdx)
}

func (p *Parser) RandChannel() int {
	p.hopIdx = rand.Intn(p.channelCount)
	return p.channelFreq(p.hopIdx)
}

func (p *Parser) ChannelIdx() int {
	return p.hopPattern[p.hopIdx]
}

func (p *Parser) ChannelPPM() int {
	if ppm, exists := p.channelPPM[p.hopPattern[p.hopIdx]]; !exists {
		return p.currentPPM
	} else {
		p.currentPPM = ppm
		return ppm
	}
}

func (p *Parser) Parse(pkts []dsp.Packet) (msgs []Message) {
	seen := make(map[string]bool)

	for _, pkt := range pkts {
		for idx, b := range pkt.Data {
			pkt.Data[idx] = SwapBitOrder(b)
		}

		s := string(pkt.Data)
		if seen[s] {
			continue
		}
		seen[s] = true

		// If the checksum fails, bail.
		if p.Checksum(pkt.Data[2:]) != 0 {
			continue
		}

		var mean float64
		symLen := p.Cfg.SymbolLength

		lower := pkt.Idx + 8*symLen
		upper := pkt.Idx + 24*symLen
		if upper > p.Cfg.BlockSize {
			upper = p.Cfg.BlockSize - 1
		}
		tail := p.Demodulator.Discriminated[lower:upper]
		for _, sample := range tail {
			mean += sample
		}
		mean /= float64(len(tail))

		freqError := 9600 + (mean*float64(p.Cfg.SampleRate))/(2*math.Pi)
		freq := float64(p.channelFreq(p.hopIdx)) / 1e6
		ppm := int(math.Floor(freqError/freq + 0.5))

		p.channelPPM[p.hopPattern[p.hopIdx]] = p.currentPPM + ppm
		p.currentPPM += ppm

		msgs = append(msgs, NewMessage(pkt))
	}

	return
}

type Message struct {
	dsp.Packet

	ID     byte
	Sensor Sensor

	WindSpeed     byte
	WindDirection byte
}

func NewMessage(pkt dsp.Packet) (m Message) {
	m.Idx = pkt.Idx
	m.Data = make([]byte, len(pkt.Data)-2)
	copy(m.Data, pkt.Data[2:])

	m.ID = m.Data[0] & 0xF
	m.Sensor = Sensor(m.Data[0] >> 4)
	m.WindSpeed = m.Data[1]
	m.WindDirection = m.Data[2]
	return m
}

func (m Message) String() string {
	return fmt.Sprintf("{ID:%d Sensor:%s WindSpeed:%d WindDir:%d}", m.ID, m.Sensor, m.WindSpeed, m.WindDirection)
}

type Sensor byte

const (
	UVIndex        Sensor = 4
	SolarRadiation Sensor = 6
	Light          Sensor = 7
	Temperature    Sensor = 8
	Humidity       Sensor = 0x0A
	Rain           Sensor = 0x0E
)

func (s Sensor) String() string {
	switch s {
	case UVIndex:
		return "UV Index"
	case SolarRadiation:
		return "Solar Radiation"
	case Light:
		return "Light"
	case Temperature:
		return "Temperature"
	case Humidity:
		return "Humidity"
	case Rain:
		return "Rain"
	default:
		return fmt.Sprintf("Unknown(0x%0X)", byte(s))
	}
}

func SwapBitOrder(b byte) byte {
	b = ((b & 0xF0) >> 4) | ((b & 0x0F) << 4)
	b = ((b & 0xCC) >> 2) | ((b & 0x33) << 2)
	b = ((b & 0xAA) >> 1) | ((b & 0x55) << 1)
	return b
}
