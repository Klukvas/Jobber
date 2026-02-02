// Toast Notifications (simple implementation)
export function showSuccessNotification(message: string): void {
  // TODO: Replace with proper toast library (e.g., react-hot-toast, sonner)
  console.log('✅ Success:', message);
  if ('Notification' in window && Notification.permission === 'granted') {
    new Notification('Success', { body: message });
  }
}

export function showErrorNotification(message: string): void {
  // TODO: Replace with proper toast library
  console.error('❌ Error:', message);
  if ('Notification' in window && Notification.permission === 'granted') {
    new Notification('Error', { body: message });
  }
}

// Push Notifications utility functions

export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (!('Notification' in window)) {
    console.warn('This browser does not support notifications');
    return 'denied';
  }

  if (Notification.permission === 'granted') {
    return 'granted';
  }

  if (Notification.permission !== 'denied') {
    const permission = await Notification.requestPermission();
    return permission;
  }

  return Notification.permission;
}

export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  if (!('serviceWorker' in navigator)) {
    console.warn('Service Workers are not supported');
    return null;
  }

  try {
    const registration = await navigator.serviceWorker.register('/sw.js');
    console.log('Service Worker registered:', registration);
    return registration;
  } catch (error) {
    console.error('Service Worker registration failed:', error);
    return null;
  }
}

export async function subscribeToPushNotifications(
  registration: ServiceWorkerRegistration,
  vapidPublicKey: string
): Promise<PushSubscription | null> {
  try {
    const applicationServerKey = urlBase64ToUint8Array(vapidPublicKey);
    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: applicationServerKey as BufferSource,
    });

    console.log('Push subscription:', subscription);
    // Send this subscription to your backend
    return subscription;
  } catch (error) {
    console.error('Failed to subscribe to push notifications:', error);
    return null;
  }
}

function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

export async function initializePushNotifications(): Promise<void> {
  // Request permission
  const permission = await requestNotificationPermission();
  if (permission !== 'granted') {
    console.log('Notification permission not granted');
    return;
  }

  // Register service worker
  const registration = await registerServiceWorker();
  if (!registration) {
    console.log('Service Worker registration failed');
    return;
  }

  // Note: In production, you would get the VAPID public key from your backend
  // and subscribe to push notifications here
  console.log('Push notifications initialized');
}
