// Copyright (c) 2024. Heusala Group Oy <info@heusalagroup.fi>. All rights reserved.

document.addEventListener('DOMContentLoaded', () => {
    const terminal = new Terminal();
    const socketUrl = `ws://${location.hostname}:${location.port}/ssh`;
    const socket = new WebSocket(socketUrl);
    const terminalContainer = document.getElementById('terminal-container');
    terminal.open(terminalContainer);

    // Handle the WebSocket connection open event
    socket.onopen = function() {
        console.log('WebSocket connected');
    };

    // Handle incoming messages from the WebSocket (SSH output)
    socket.onmessage = function(event) {
    terminal.write(event.data);
};

    // Send data to the WebSocket (SSH input)
    terminal.onData(data => {
        console.log(data);
    
        // Ensure the WebSocket is in an OPEN state before sending data
        if (socket.readyState === WebSocket.OPEN) {
            socket.send(data);
        } else {
            console.error('WebSocket is not open. ReadyState:', socket.readyState);
        }
    });

    // Handle the WebSocket close event
    socket.onclose = function() {
        console.log('WebSocket disconnected');
        terminal.write('\r\n\x1B[1;31mDisconnected from SSH server.\x1B[0m\r\n');
    };
});
