const websocketUrl = "ws://localhost:9001";
let websocket = createConn(websocketUrl);

/**
 * @type {HTMLUListElement}
 */
const messagesList = document.getElementById("messages");
/**
 * @type {HTMLUListElement}
 */
const statusUpdatesList = document.getElementById("status-updates");

/**
 * @type {HTMLInputElement}
 */
const messageInput = document.getElementById("message-input");
/**
 * @type {HTMLButtonElement}
 */
const sendMessageButton = document.getElementById("send-message");

/**
 * @type {HTMLButtonElement}
 */
const toggleConnectionButton = document.getElementById("toggle-connection");
/**
 * 
 * @type {HTMLButtonElement}
 */
const clearButton = document.getElementById("clear");

function msg(s) {
  const now = new Date();
  return `${now.getUTCHours()}:${now.getUTCMinutes()}:${now.getUTCSeconds()}.${now.getUTCMilliseconds()} - ${s}`
}

function createConn(url) {
  const websocket = new WebSocket(websocketUrl);

  websocket.addEventListener("open", () => {
    addListItem(statusUpdatesList, msg("connection opened"))
  })

  websocket.addEventListener("close", () => {
    addListItem(statusUpdatesList, msg("connection closed"))
  })

  websocket.addEventListener("error", () => {
    addListItem(statusUpdatesList, msg("connection error"))
  })

  websocket.addEventListener("message", (event) => {
    addListItem(messagesList, msg(event.data));
  })

  return websocket;
}

sendMessageButton?.addEventListener('click', () => {
  const message = messageInput?.value
  websocket.send(message ?? 'null_message')
  // messageInput.value = '';
})

toggleConnectionButton?.addEventListener('click', () => {
  if (websocket.readyState === WebSocket.CLOSED) {
    websocket = createConn(websocketUrl);
  } else {
    websocket.close();
  }
})

clearButton?.addEventListener('click', () => {
  statusUpdatesList.innerHTML = ''
  messagesList.innerHTML = ''
})

/**
 * @param {HTMLUListElement} list 
 * @param {string} text
 */
function addListItem(list, text) {
  const li = document.createElement("li");
  list.prepend(li);
  li.innerText = text;
}
