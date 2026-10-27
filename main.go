package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"slices"
	"strings"
	"sync"

	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/bigtable/bttest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	project := getenv("BIGTABLE_PROJECT", "dev")
	instance := getenv("BIGTABLE_INSTANCE", "dev")

	schema, err := parseSchema(os.Getenv("BIGTABLE_SCHEMA"))
	if err != nil {
		log.Fatal(err)
	}

	addr, err := start(getenv("BIGTABLE_EMULATOR_HOST", "0.0.0.0:8086"), project, instance, schema)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Cloud Bigtable emulator running on %s with %d tables", addr, len(schema))

	select {}
}

// start creates the tables in schema before it opens addr, so a client that can connect always finds them.
func start(addr, project, instance string, schema map[string][]string) (net.Addr, error) {
	internal, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	listener := &listeners{addr: internal.Addr(), conns: make(chan net.Conn), closed: make(chan struct{})}
	listener.accept(internal)

	if _, err := bttest.NewServerWithListener(listener); err != nil {
		return nil, err
	}

	if err := createTables(internal.Addr().String(), project, instance, schema); err != nil {
		return nil, err
	}

	public, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	listener.accept(public)

	return public.Addr(), nil
}

func createTables(addr, project, instance string, schema map[string][]string) error {
	ctx := context.Background()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	admin, err := bigtable.NewAdminClient(ctx, project, instance, option.WithGRPCConn(conn))
	if err != nil {
		return err
	}
	defer admin.Close()

	for table, families := range schema {
		if err := admin.CreateTable(ctx, table); err != nil {
			return fmt.Errorf("creating table %s: %w", table, err)
		}

		for _, family := range families {
			if err := admin.CreateColumnFamily(ctx, table, family); err != nil {
				return fmt.Errorf("creating column family %s in table %s: %w", family, table, err)
			}
		}
	}

	return nil
}

// parseSchema reads `<table>:<family>#<family>,<table>:<family>`, merging the families of a table listed twice.
func parseSchema(value string) (map[string][]string, error) {
	schema := map[string][]string{}

	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		table, families, _ := strings.Cut(entry, ":")
		if table == "" {
			return nil, fmt.Errorf("BIGTABLE_SCHEMA entry %q has no table name", entry)
		}

		if _, ok := schema[table]; !ok {
			schema[table] = []string{}
		}

		for _, family := range strings.Split(families, "#") {
			if family != "" && !slices.Contains(schema[table], family) {
				schema[table] = append(schema[table], family)
			}
		}
	}

	return schema, nil
}

// listeners is a net.Listener that accepts connections from every listener handed to accept.
type listeners struct {
	addr      net.Addr
	conns     chan net.Conn
	closed    chan struct{}
	closeOnce sync.Once
}

func (l *listeners) accept(listener net.Listener) {
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			select {
			case l.conns <- conn:
			case <-l.closed:
				conn.Close()
				return
			}
		}
	}()
}

func (l *listeners) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *listeners) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

func (l *listeners) Addr() net.Addr { return l.addr }

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
