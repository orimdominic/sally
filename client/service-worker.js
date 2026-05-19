// Register event listener for the 'push' event.
self.addEventListener("push", function (event) {
	const payload = event.data.text() ?? `{"title":"Notification","body":"Notification received"}`;
  const notif = JSON.parse(payload)
  // Keep the service worker alive until the notification is created.
  event.waitUntil(
    // Show a notification with title 'ServiceWorker Cookbook' and body 'Alea iacta est'.
    self.registration.showNotification(notif.title, {
      body: notif.body,
    }),
  );
});
