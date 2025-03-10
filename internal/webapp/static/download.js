function downloadCSV(url) {
    // Create a hidden form and submit it
    const form = document.createElement('form')
    form.method = 'GET';
    form.action = url;
    form.style.display = 'none';
    document.body.appendChild(form);
    form.submit();
    document.body.removeChild(form);
}
