
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
    console.log("running websocket");
    const ws = new WebSocket(`ws://localhost:8081/job/${window.jobStr}/stream`);

    ws.addEventListener("message", (event) => {
      const div = document.createElement("div");
      div.appendChild(document.createTextNode(event.data));
      container.appendChild(div);
    });

    ws.addEventListener("close", (event) => {
      const div = document.createElement("div");
      div.appendChild(document.createTextNode("Web socket closed."));
      container.appendChild(div);
    });
  }
};

document.addEventListener("DOMContentLoaded", () => {
  console.log(`main.js executing`);
  alerts();
  webSocket();
});
