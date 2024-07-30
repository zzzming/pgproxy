package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/zzzming/pgproxy/pkg/config"
)

// Postgres systerm values
const (
	AuthRequestCode         = 0x0000000E
	SSLRequestCode          = 80877103
	protocolVersion         = 196608
	applicationName         = "application_name"
	fallbackApplicationName = "fallback_application_name"
	options                 = "options"
)

func parseStartupMessage(message []byte) map[string]string {
	params := make(map[string]string)

	// Skip protocol version (first 4 bytes)
	for i := 8; i < len(message)-1; {
		key := ""
		value := ""

		// Read key
		for j := i; j < len(message); j++ {
			if message[j] == 0 {
				key = string(message[i:j])
				i = j + 1
				break
			}
		}

		// Read value
		for j := i; j < len(message); j++ {
			if message[j] == 0 {
				value = string(message[i:j])
				i = j + 1
				break
			}
		}

		if key != "" && value != "" {
			params[key] = value
		}
		fmt.Printf("%s : %s\n", key, value)

		if i >= len(message) || message[i] == 0 {
			break
		}
	}

	return params
}

func requestPassword(conn net.Conn) error {
	authRequest := []byte{'R', 0, 0, 0, 8, 0, 0, 0, 3}
	_, err := conn.Write(authRequest)
	return err
}

func readPasswordMessage(conn net.Conn) ([]byte, error) {
	msgTypeBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, msgTypeBuf); err != nil {
		return nil, err
	}

	if msgTypeBuf[0] != 'p' {
		return nil, fmt.Errorf("expected password message, got %c", msgTypeBuf[0])
	}

	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}
	messageLen := int(binary.BigEndian.Uint32(lenBuf)) - 4

	passwordBuf := make([]byte, messageLen)
	if _, err := io.ReadFull(conn, passwordBuf); err != nil {
		return nil, err
	}

	return append(msgTypeBuf, passwordBuf...), nil
}

func HandleConnection(conn net.Conn, cfg *config.Config) error {
	// logger, _ : zap.NewProduction()
	// defer logger.Sync()
	defer conn.Close()

	// Read message length in the first four bytes
	lenBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, lenBuf)
	if err != nil {
		return fmt.Errorf("reading message length error: %v", err)
	}
	messageLen := int(binary.BigEndian.Uint32(lenBuf))

	// check protocol version
	// only support PostgreSQL 3.0
	message := make([]byte, messageLen)
	copy(message, lenBuf) // Include the length bytes in the message
	_, err = io.ReadFull(conn, message[4:])
	if err != nil {
		return fmt.Errorf("reading message error: %v", err)
	}

	// Print raw bytes for debugging
	fmt.Printf("Raw message: %v\n", message)

	// Check protocol version (should be 196608 for PostgreSQL 3.0)
	requestedVersion := binary.BigEndian.Uint32(message[4:8])
	if requestedVersion != protocolVersion {
		// TODO: SSL is not supportet yet!!!
		conn.Write([]byte{'N'}) // rejecting SSL and any other version
	}

	// Parse startup parameters
	fmt.Println("Startup parameters:")

	params := parseStartupMessage(message)
	appName := params["application_name"]
	fmt.Printf("Application name: %s\n", appName)

	// Request password from client
	if err := requestPassword(conn); err != nil {
		return fmt.Errorf("requesting password:", err)
	}

	// Read password message from client
	passwordMessage, err := readPasswordMessage(conn)
	if err != nil {
		return fmt.Errorf("reading password message:", err)
	}

	password := string(passwordMessage[1 : len(passwordMessage)-1]) // Remove message type byte and null terminator
	// Construct the connection string for the backend
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		params["user"],
		password,
		cfg.TargetHost,
		cfg.TargetPort,
		params["database"])

	fmt.Println(connString)

	// Connect to the backend PostgreSQL server
	backendConn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return fmt.Errorf("connecting to backend database:", err)
	}
	defer backendConn.Close(context.Background())

	// Proxy data between client and backend
	proxyData(conn, backendConn, appName)

	return nil
}

func proxyData(clientConn net.Conn, backendConn *pgx.Conn, appName string) {
	buffer := make([]byte, 4096)

	for {
		n, err := clientConn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading from client:", err)
			}
			break
		}

		query := string(buffer[:n])
		modifiedQuery := addNamespaceToQuery(query, appName)

		rows, err := backendConn.Query(context.Background(), modifiedQuery)
		if err != nil {
			fmt.Println("Error executing query on backend:", err)
			continue
		}

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				fmt.Println("Error getting row values:", err)
				continue
			}
			fmt.Fprintf(clientConn, "%v\n", values)
		}
		rows.Close()
	}
}

func addNamespaceToQuery(query, namespace string) string {
	modifiedQuery := strings.ReplaceAll(query, " FROM ", fmt.Sprintf(" FROM %s.", namespace))
	modifiedQuery = strings.ReplaceAll(modifiedQuery, " JOIN ", fmt.Sprintf(" JOIN %s.", namespace))
	return modifiedQuery
}
