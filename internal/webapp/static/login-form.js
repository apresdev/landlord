document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('login-form');
    if (form === null) {
        return;
    }
    const submitButton = document.getElementById('login-submit-button');
    if (submitButton === null) {
        return;
    }
    const spinner = document.getElementById("login-spinner");
    if (spinner === null) {
        return;
    }
    form.addEventListener('submit', function(e) {
        spinner.style.opacity = '1';
        submitButton.disabled = true;
    });
});
