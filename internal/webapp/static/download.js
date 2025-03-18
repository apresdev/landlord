function downloadInBackground(url, buttonId) {
    // Get the button element
    const button = document.getElementById(buttonId) || this;

    // Disable the button
    button.originalText = button.textContent;
    button.textContent = "Preparing report...";

    // Create a hidden form for the download
    const form = document.createElement('form');
    form.method = 'GET';
    form.action = url;
    form.style.display = 'none';
    document.body.appendChild(form);

    // Submit the form to start the download
    form.submit();

    // Since we can't reliably detect when a download completes,
    // re-enable the button after a reasonable timeout
    setTimeout(function() {
        button.textContent = button.originalText;

        // Clean up the form
        document.body.removeChild(form);
    }, 5000);
}
