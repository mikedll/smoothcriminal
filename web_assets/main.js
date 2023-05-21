
const alerts = () => {
  if(window.error !== '') {
    const alerts = document.querySelector('.alerts-container');
    const alert = document.createElement('div');
    alert.appendChild(document.createTextNode(error));
    alert.classList.add('alert');
    alert.classList.add('alert-danger');
    alerts.appendChild(alert);
  }
}  

const webSocket = () => {
  const container = document.querySelector('.job-container');

  if(container !== null) {
    const addMessage = (m) => {
      const div = document.createElement("div");
      div.appendChild(document.createTextNode(m));
      container.appendChild(div);
    }

    const pathRegexp = RegExp("/jobs/(\\d+)")
    const matches = location.pathname.match(pathRegexp);
    if(matches === null) {
      addMessage(`Unable to parse path from: ${location.pathname}`);
    }

    const ws = new WebSocket(`ws://localhost:8081/jobs/${matches[1]}/stream`);

    ws.addEventListener("open", (event) => {
      addMessage("Web socket connection opened");
    });

    ws.addEventListener("message", (event) => {
      addMessage(event.data);
    });

    ws.addEventListener("close", (event) => {
      addMessage("Web socket connection closed.");
    });
  }
};

document.addEventListener("DOMContentLoaded", () => {
  console.log(`main.js executing`);
  alerts();
  webSocket();
});
