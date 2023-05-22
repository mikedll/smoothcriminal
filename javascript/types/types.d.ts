
interface Subscription {
  name: string;
}

interface MessageJobStatus {
  type: "message";
  message: string;
}

interface PercentJobStatus {
  type: "complete"
  percentComplete: number;
}