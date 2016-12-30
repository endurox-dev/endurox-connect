package main

import (
	"bufio"
	"fmt"
	"net"
	"runtime"
	"strings"
	"ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

type NConn struct {
	conn net.Conn /* have a connection handler... */
	id   int64
}

type InMsg struct {
	data string
	id   int64
}

type Client struct {
	incoming chan InMsg
	outgoing chan string
	reader   *bufio.Reader
	writer   *bufio.Writer
	ncon     NConn /* have a connection handler... */
}

type LeSrv struct {
	/*clients []*Client */
	clients  map[int64]*Client
	joins    chan NConn
	incoming chan InMsg
	outgoing chan string
}

var M_leSrv *LeSrv
var M_id int64 /* TODO: think about wrapping around...! */

func (client *Client) Read() {
	for {
		line, err := client.reader.ReadString('\n')
		if nil != err {
			/* We should remove ourselves from array */
			/* if conn object existed, it will
			 * remove client from arr
			 */
			M_leSrv.joins <- client.ncon
			/* Kill the chan */
			client.outgoing <- ""
			client.ncon.conn.Close()
			client.ncon.conn = nil
			break /* finish it. */
		} else {
			msg := InMsg{data: line, id: client.ncon.id}
			client.incoming <- msg
		}
	}
	fmt.Printf("in chan exit...\n")
}

func (client *Client) Write() {
	for data := range client.outgoing {
		/* Proces stop on channel... */
		if data == "" {
			break
		} else {
			client.writer.WriteString(data)
			client.writer.Flush()
		}
	}
	fmt.Printf("out chan exit...\n")
}

func (client *Client) Listen() {
	go client.Read()
	go client.Write()
}

func NewClient(ncon NConn) *Client {
	writer := bufio.NewWriter(ncon.conn)
	reader := bufio.NewReader(ncon.conn)

	fmt.Printf("Added client: [%d]", ncon.id)

	client := &Client{
		incoming: make(chan InMsg),
		outgoing: make(chan string),
		reader:   reader,
		writer:   writer,
		ncon:     ncon,
	}

	client.Listen()

	return client
}

func TrimSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

//
// We shall lock to os thread here...
//
func (leSrv *LeSrv) ProcessRequest(msg InMsg) {

	//We shall not interrupt ATMI processing.
	runtime.LockOSThread()

	data := msg.data

	data = TrimSuffix(data, "\n")

	fmt.Printf("Got line: [%s]\n", data)

	buf, err := atmi.NewUBF(1024)

	if err != nil {
		fmt.Printf("ATMI Error %d:[%s]\n", err.Code(), err.Message())
		return
	}

	/* parse the buffer */
	sbuf := strings.Split(data, "|")

	for _, keyval := range sbuf {
		field := strings.SplitN(keyval, "=", 2)
		if len(field) > 1 {
			/* Load the protocol field */
			fmt.Printf("got field name [%s] value [%s]\n",
				field[0], field[1])

			id, err := atmi.BFldId(field[0])

			if nil != err {
				fmt.Printf("Unknown field - %d:[%s]\n", err.Code(), err.Message())
			} else if err := buf.BAdd(id, field[1]); nil != err {
				fmt.Printf("UBF Error %d:[%s]\n", err.Code(), err.Message())
			}
		} else {
			fmt.Printf("ERROR! Cannot parse field [%s]\n", field)
		}
	}

	/* Dump the buffer */
	buf.BPrint()

	if err := buf.BAdd(ubftab.L_CONID, msg.id); nil != err {
		fmt.Printf("Failed to set L_CONID - UBF Error %d:[%s]\n",
			err.Code(), err.Message())
		return
	}

	if err := buf.BAdd(ubftab.L_CONGW, "LTCP"); nil != err {
		fmt.Printf("Failed to set L_CONGW - UBF Error %d:[%s]\n",
			err.Code(), err.Message())
		return
	}

	/* if we have CMD id, try to call the service */
	svcnm, err := buf.BGetString(ubftab.L_CMD, 0)
	if err != nil {
		fmt.Printf("Missing L_CMD - drop the packet... %d:[%s]\n", err.Code(), err.Message())
	} else {

		fmt.Printf("Calling service [%s]\n", svcnm)

		if _, err := atmi.TpCall(svcnm, buf, 0); nil != err {
			fmt.Printf("ATMI Error %d:[%s]\n", err.Code(), err.Message())
		}
	}
}

func (leSrv *LeSrv) Join(ncon NConn) {

	if nil != leSrv.clients[ncon.id] {
		fmt.Printf("Killing client %d\n", ncon.id)
		leSrv.clients[ncon.id] = nil
	} else {

		client := NewClient(ncon)
		leSrv.clients[ncon.id] = client
		/* Process client incomings... */
		go func() {
			for {
				leSrv.incoming <- <-client.incoming
			}
		}()
	}
}

func (leSrv *LeSrv) Listen() {
	go func() {
		for {
			select {
			case data := <-leSrv.incoming:
				leSrv.ProcessRequest(data)
			case ncon := <-leSrv.joins:
				leSrv.Join(ncon)
			}
		}
	}()
}

func NewLeSrv() *LeSrv {
	leSrv := &LeSrv{
		clients:  make(map[int64]*Client),
		joins:    make(chan NConn),
		incoming: make(chan InMsg),
		outgoing: make(chan string),
	}

	leSrv.Listen()

	return leSrv
}

func NetRun() {
	M_leSrv = NewLeSrv()

	listener, _ := net.Listen("tcp", ":9973")

	for {
		conn, _ := listener.Accept()
		var ncon NConn
		/* todo: wrapping & checking of existing conn & reject if full */
		M_id++
		ncon.id = M_id
		ncon.conn = conn
		M_leSrv.joins <- ncon
	}
}
