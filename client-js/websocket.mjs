
const acceptedUrlProtocolsMap = {
  ws: "ws:",
  wss: "wss:",
  http: "http:",
  https: "https:",
};
const acceptedUrlProtocols = Object.values(acceptedUrlProtocolsMap);

export class UWebSocket {
  /**
   * @type {URL}
   */
  #url

  /**
  * @type {Array<string>}
  */
  #protocols

  /**
  * @type {TextEncoder}
  */
  #textEncoder

  /**
   * @param {string | URL} url
   * @param {string | Array<string> | undefined} protocols
   */
  constructor(url, protocols) {
    if (typeof url === "string") {
      this.#url = new URL(url);
    } else if (url instanceof URL) {
      this.#url = url;
    } else {
      throw new TypeError("url must be type of string or URL")
    }

    if (!acceptedUrlProtocols.includes(this.#url.protocol)) {
      throw new Error(`${this.#url.protocol} is not in the list of accepted protocols: ${acceptedUrlProtocols}`)
    }

    if (typeof protocols === "string") {
      this.#protocols = [protocols];
    } else if (Array.isArray(protocols)) {
      const isStringArray = protocols.every(v => typeof v === "string");
      if (!isStringArray) {
        throw new TypeError("if protocols is an array, it must only contain strings")
      }
      this.#protocols = protocols;
    } else {
      this.#protocols = [];
    }

    this.#textEncoder = new TextEncoder();

    this.#handshake();
  }

  send(data) { }

  close() { }

  async #handshake() {
    console.log("handshake");

    const secKey = btoa(crypto.randomUUID());
    const expectedSecAccept = await this.#secAcceptFromSecKey(secKey)

    /**
     * @type {HeadersInit}
     */
    const headers = {
      "Upgrade": "websocket",
      "Connection": "Upgrade",
      "Sec-WebSocket-Key": secKey,
      "Sec-WebSocket-Version": "13",
    };
    if (this.#protocols.length !== 0) {
      headers["Sec-WebSocket-Protocol"] = this.#protocols.join(", ")
    }

    const handshakeUrl = new URL(this.#url.toString());
    if (handshakeUrl.protocol === acceptedUrlProtocolsMap.ws) {
      handshakeUrl.protocol = acceptedUrlProtocolsMap.http;
    } else if (handshakeUrl.protocol === acceptedUrlProtocolsMap.wss) {
      handshakeUrl.protocol = acceptedUrlProtocolsMap.https;
    }

    const response = await fetch(handshakeUrl, {
      method: "GET",
      headers: headers,
    });

    if (response.status !== 101) {
      console.log(response);
      console.log(await response.text())
      throw new Error("Error during handshake, returned status is not 101")
    }

    if (response.headers.get("Upgrade") !== "websocket") {
      throw new Error("Error during handshake, Upgrade header does not equal websocket")
    }

    if (response.headers.get("Connection") !== "Upgrade") {
      throw new Error("Error during handshake, Connection header does not equal Upgrade")
    }

    const secAccept = response.headers.get("Sec-WebSocket-Accept")
    if (!secAccept) {
      throw new Error("Error during handshake Sec-WebSocket-Accept header is not present")
    }
    if (secAccept !== expectedSecAccept) {
      throw new Error("Error during handshake Sec-WebSocket-Accept header does not equal expected value")
    }

    console.log("handshake ok");
  }

  async #secAcceptFromSecKey(secKey) {
    const guid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

    const concat = secKey + guid
    const data = this.#textEncoder.encode(concat)
    const hash = await crypto.subtle.digest("SHA-1", data)
    const b64 = btoa(hash)

    return b64
  }
}
