// Register a Service Worker.
if (!("serviceWorker" in navigator)) {
	alert("PWA not supported");
}

let sub = null;

// triggers navigator.serviceWorker.ready when done
navigator.serviceWorker.register("service-worker.js");

navigator.serviceWorker.ready.then(async function (registration) {
	const subscription = await registration.pushManager.getSubscription();
	if (subscription) {
		sub = subscription;
		return subscription;
	}

	const response = await fetch("/publickey", {
		method: "GET",
	});
	const { publicKey } = await response.json();
	const convertedVapidKey = urlBase64ToUint8Array(publicKey);
	return await registration.pushManager.subscribe({
		userVisibleOnly: true,
		applicationServerKey: convertedVapidKey,
	});
});

// This function is needed because Chrome doesn't accept a base64 encoded string
// as value for applicationServerKey in pushManager.subscribe yet
// https://bugs.chromium.org/p/chromium/issues/detail?id=802280
function urlBase64ToUint8Array(base64String) {
	var padding = "=".repeat((4 - (base64String.length % 4)) % 4);
	var base64 = (base64String + padding).replace(/\-/g, "+").replace(/_/g, "/");

	var rawData = window.atob(base64);
	var outputArray = new Uint8Array(rawData.length);

	for (var i = 0; i < rawData.length; ++i) {
		outputArray[i] = rawData.charCodeAt(i);
	}

	return outputArray;
}

document
	.getElementById("pdfUploadForm")
	.addEventListener("submit", async function (e) {
		e.preventDefault();

		const fileInput = document.getElementById("pdfFile");
		const statusMessage = document.getElementById("statusMessage");

		if (fileInput.files.length === 0) {
			statusMessage.style.color = "red";
			statusMessage.textContent = "Please select a file first.";
			return;
		}

		const file = fileInput.files[0];
		if (file.type !== "application/pdf") {
			statusMessage.style.color = "red";
			statusMessage.textContent = "Only PDF files are allowed.";
			return;
		}

		const formData = new FormData();
		formData.append("file", file);
		formData.append("subscription", JSON.stringify(sub));

		statusMessage.style.color = "blue";
		statusMessage.textContent = "Uploading...";

		try {
			const backendUrl = "/documents";

			const response = await fetch(backendUrl, {
				method: "POST",
				body: formData,
			});

			if (response.ok) {
				statusMessage.style.color = "green";
				statusMessage.textContent =
					"PDF uploaded successfully! You will get a notification when the embedding is completed";
			} else {
				statusMessage.style.color = "red";
				statusMessage.textContent = `Upload failed with status: ${response.status}`;
			}
		} catch (error) {
			statusMessage.style.color = "red";
			statusMessage.textContent = "An error occurred during upload.";
			console.error("Error:", error);
		}
	});

document
	.getElementById("sentenceForm")
	.addEventListener("submit", async function (e) {
		e.preventDefault();

		const sentenceInput = document.getElementById("sentenceInput");
		const sentenceList = document.getElementById("sentenceList");
		const submitButton = this.querySelector('button[type="submit"]');
		const rawTextValue = sentenceInput.value.trim();

		if (!rawTextValue) return;

		submitButton.disabled = true;
		submitButton.textContent = "Fetching...";

		sentenceList.innerHTML = "";

		try {
			const queryParams = new URLSearchParams({ query: rawTextValue });

			const targetUrl = `/query?${queryParams.toString()}`;

			const response = await fetch(targetUrl, {
				method: "GET",
			});

			if (!response.ok) {
				throw new Error(`Server responded with status: ${response.status}`);
			}

			const { results } = await response.json();

			if (Array.isArray(results)) {
				if (results.length === 0) {
					const noResults = document.createElement("li");
					noResults.textContent = "No matching sentences found.";
					noResults.style.borderLeftColor = "#cca200";
					sentenceList.appendChild(noResults);
				} else {
					results.forEach((sentence) => {
						const listItem = document.createElement("li");
						listItem.textContent = sentence;
						sentenceList.appendChild(listItem);
					});
				}
			} else {
				console.error("Expected an array from API but received:", results);
			}
		} catch (error) {
			console.error("API Error:", error);

			// Display a quick inline error item inside the list to alert the user
			const errorItem = document.createElement("li");
			errorItem.textContent = "❌ Failed to fetch sentences from server.";
			errorItem.style.borderLeftColor = "#dc2626"; // Red error color
			sentenceList.appendChild(errorItem);
		} finally {
			// 6. Reset button back to its original state
			submitButton.disabled = false;
			submitButton.textContent = "Search";
		}
	});
