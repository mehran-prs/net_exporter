// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"os"
	"testing"
)

func TestParseTcpFile(t *testing.T) {

	//testNetStats(t, "fixtures/proc/net/netstat")
}

func TestParseAsnRecords(t *testing.T) {
	f, err := os.Open("fixtures/asn.csv")
	if err != nil {
		t.Fatal(err)
	}

	l, err := parseAsnRecords(f)
	if err != nil {
		t.Fatal(err)
	}

	if want, got := 4, len(l); want != got {
		t.Errorf("wants %d records, got %d", want, got)
	}

	// Check first record:

	got := l[0]

	if got.IPNet.String() != "2.63.144.0/20" {
		t.Errorf("first record is not valid, wants %s, got %s", "2.63.144.0/20", got.IPNet.String())
	}

	if got.Name != "Miranda-Media Ltd" {
		t.Errorf("first record is not valid, wants %s, got %s", "Miranda-Media Ltd", got.Name)
	}
}

//func testNetStats(t *testing.T, fileName string) {
//	file, err := os.Open(fileName)
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer file.Close()
//
//	netStats, err := parseTCPStats(file, fileName)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if want, got := "102471", netStats["TcpExt"]["DelayedACKs"]; want != got {
//		t.Errorf("want netstat TCP DelayedACKs %s, got %s", want, got)
//	}
//
//	if want, got := "2786264347", netStats["IpExt"]["OutOctets"]; want != got {
//		t.Errorf("want netstat IP OutOctets %s, got %s", want, got)
//	}
//}
