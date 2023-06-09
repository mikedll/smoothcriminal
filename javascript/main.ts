
interface Window {
  error: string;
  subscriptions: string;
  host: string;
}

const alerts = () => {
  if(window.error !== '') {
    const alerts = document.querySelector('.alerts-container');
    if(alerts === null) {
      console.error("unable to find alerts container");
      return;
    }
    const alert = document.createElement('div');
    alert.appendChild(document.createTextNode(window.error));
    alert.classList.add('alert');
    alert.classList.add('alert-danger');
    alerts.appendChild(alert);
  }
}

const subscriptions = () => {
  const ul = document.querySelector('.subscriptions ul');
  
  if(ul === null) return;

  const subscriptions: Subscription[] = JSON.parse(window.subscriptions);
  subscriptions.forEach((s: Subscription) => {
    const li = document.createElement('li');
    li.appendChild(document.createTextNode(s.name));
    ul.appendChild(li);
  });

  const summary = document.createElement('p');
  summary.appendChild(document.createTextNode(`Found ${subscriptions.length} subscription(s).`))
  const container = ul.closest('div');
  if(container === null) {
    console.error("unable to find container div of ul");
    return;
  }
  container.appendChild(summary);
}

const webSocket = () => {
  const container = document.querySelector('.job-container');
  
  if(container === null) 
    return;
    
  const messages = container.querySelector('.messages')!;
  const progressBar: HTMLDivElement = container.querySelector('.progress-bar')!;

  const addMessage = (m: string) => {
    const div = document.createElement("div");
    div.appendChild(document.createTextNode(m));
    messages.appendChild(div);
  }

  const pathRegexp = RegExp("/jobs/(\\d+)")
  const matches = location.pathname.match(pathRegexp);
  if(matches === null) {
    addMessage(`Unable to parse path from: ${location.pathname}`);
    return;
  }

  const ws = new WebSocket(`ws://${window.host}/jobs/${matches[1]}/stream`);

  ws.addEventListener("open", (event) => {
    addMessage("Web socket connection opened");
  });

  ws.addEventListener("message", (event) => {
    const jobStatus: MessageJobStatus | PercentJobStatus = JSON.parse(event.data);
    switch(jobStatus.type) {
      case "message":
        addMessage(jobStatus.message);
        break;
      case "complete":
        const wholeNum = Math.round(jobStatus.percentComplete * 100);
        progressBar.style.width = `${wholeNum}%`;
        break;
    }
  });

  ws.addEventListener("close", (event) => {
    addMessage("Web socket connection closed.");
  });
};

document.addEventListener("DOMContentLoaded", () => {
  console.log(`main.js executing`);
  alerts();
  webSocket();
  subscriptions();
});
