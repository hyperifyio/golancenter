package main

import (
    "log"
    "net/http"

    "github.com/gorilla/websocket"
    "golang.org/x/crypto/ssh"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow connections from any origin.
    },
}

func sshDial() (*ssh.Session, error) {
    config := &ssh.ClientConfig{
        User: "Seeti Oinonen", // username
        Auth: []ssh.AuthMethod{
            ssh.Password("seesti200"), // password
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For testing purposes only
    }

    conn, err := ssh.Dial("tcp", "localhost:22", config)
    if err != nil {
        return nil, err
    }

    session, err := conn.NewSession()
    if err != nil {
        return nil, err
    }
    return session, nil
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
    wsConn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket Upgrade error:", err)
        return
    }
    defer wsConn.Close()

    session, err := sshDial()
    if err != nil {
        log.Println("SSH dial error:", err)
        return
    }
    defer session.Close()

    sshIn, err := session.StdinPipe()
    if err != nil {
        log.Println("Failed to get SSH stdin pipe:", err)
        return
    }
    defer sshIn.Close()

    sshOut, err := session.StdoutPipe()
    if err != nil {
        log.Println("Failed to get SSH stdout pipe:", err)
        return
    }

    if err := session.RequestPty("xterm", 80, 40, ssh.TerminalModes{}); err != nil {
        log.Println("Failed to request pseudo terminal:", err)
        return
    }

    if err := session.Shell(); err != nil {
        log.Println("Failed to start shell:", err)
        return
    }

    go func() {
        defer sshIn.Close()
        for {
            _, message, err := wsConn.ReadMessage()
            if err != nil {
                log.Println("Error reading from WebSocket:", err)
                return
            }
            if _, err := sshIn.Write(message); err != nil {
                log.Println("Error writing to SSH stdin:", err)
                return
            }
        }
    }()

    go func() {
        buffer := make([]byte, 1024)
        for {
            n, err := sshOut.Read(buffer)
            if err != nil {
                log.Println("Error reading from SSH stdout:", err)
                return
            }
            if err := wsConn.WriteMessage(websocket.TextMessage, buffer[:n]); err != nil {
                log.Println("Error sending message to WebSocket:", err)
                return
            }
        }
    }()

    // Keep the handler alive until the WebSocket connection is closed
    for {
        if _, _, err := wsConn.NextReader(); err != nil {
            wsConn.Close()
            break
        }
    }
}
