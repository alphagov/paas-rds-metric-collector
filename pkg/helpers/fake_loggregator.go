// Note: Copied from https://github.com/cloudfoundry/go-loggregator/blob/8ebcfd3c7377510fe5a45ded5110d7749b562606/servers_test.go

package helpers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
)

type FakeLoggregatorIngressServer struct {
	Receivers    chan loggregator_v2.Ingress_BatchSenderServer
	SendReciever chan *loggregator_v2.EnvelopeBatch
	Addr         string
	tlsConfig    *tls.Config
	grpcServer   *grpc.Server
	grpc.Stream
}

func NewFakeLoggregatorIngressServer(serverCert, serverKey, caCert string) (*FakeLoggregatorIngressServer, error) {
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientAuth:         tls.RequestClientCert,
		InsecureSkipVerify: false,
	}
	caCertBytes, err := ioutil.ReadFile(caCert)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertBytes)
	tlsConfig.RootCAs = caCertPool

	return &FakeLoggregatorIngressServer{
		tlsConfig:    tlsConfig,
		Receivers:    make(chan loggregator_v2.Ingress_BatchSenderServer),
		SendReciever: make(chan *loggregator_v2.EnvelopeBatch, 100),
		Addr:         "localhost:0",
	}, nil
}

func (*FakeLoggregatorIngressServer) Sender(srv loggregator_v2.Ingress_SenderServer) error {
	return nil
}

func (t *FakeLoggregatorIngressServer) BatchSender(srv loggregator_v2.Ingress_BatchSenderServer) error {
	t.Receivers <- srv

	<-srv.Context().Done()

	return nil
}

func (t *FakeLoggregatorIngressServer) Send(_ context.Context, b *loggregator_v2.EnvelopeBatch) (*loggregator_v2.SendResponse, error) {
	t.SendReciever <- b
	return &loggregator_v2.SendResponse{}, nil
}

func (t *FakeLoggregatorIngressServer) Start() error {
	listener, err := net.Listen("tcp4", t.Addr)
	if err != nil {
		return err
	}
	t.Addr = listener.Addr().String()

	var opts []grpc.ServerOption
	if t.tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(t.tlsConfig)))
	}
	t.grpcServer = grpc.NewServer(opts...)

	loggregator_v2.RegisterIngressServer(t.grpcServer, t)

	go t.grpcServer.Serve(listener)

	return nil
}

func (t *FakeLoggregatorIngressServer) Stop() {
	t.grpcServer.Stop()
}

func (t *FakeLoggregatorIngressServer) GetEnvelopes() ([]*loggregator_v2.Envelope, error) {
	var recv loggregator_v2.Ingress_BatchSenderServer

	select {
	case recv = <-t.Receivers:
	case <-time.After(10 * time.Second):
		return []*loggregator_v2.Envelope{}, fmt.Errorf("Timeout reading for envelopes")
	}

	envBatch, err := recv.Recv()
	if err != nil {
		return []*loggregator_v2.Envelope{}, err
	}

	return envBatch.Batch, nil
}
