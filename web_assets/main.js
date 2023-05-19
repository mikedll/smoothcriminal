
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

document.addEventListener("DOMContentLoaded", () => {
  console.log(`main.js executing`);
  alerts();
});
