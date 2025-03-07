// Handlers for toggling the colour scheme

const landlordColourSchemeKey = 'landlord-colour-scheme';

document.addEventListener('DOMContentLoaded', () => {
    // Check for saved theme preference in localStorage
    let scheme = localStorage.getItem(landlordColourSchemeKey)
    if (!scheme) {
        const prefersDarkScheme = window.matchMedia('(prefers-color-scheme: dark)');
        scheme = prefersDarkScheme.matches ? 'dark' : 'light';
        localStorage.setItem(landlordColourSchemeKey, scheme);
    }

    applyColourScheme(scheme);

    function onClickToggleButton() {
        const isDarkMode = document.documentElement.classList.contains('theme-dark');
        const newScheme = isDarkMode ? 'light' : 'dark';
        localStorage.setItem(landlordColourSchemeKey, newScheme);
        applyColourScheme(newScheme);
    }

    const colourSchemeToggleLight = document.getElementById('colour-scheme-toggle-light');
    if (colourSchemeToggleLight) {
        colourSchemeToggleLight.addEventListener('click', () => {
            onClickToggleButton()
        });
    }
    const colourSchemeToggleDark = document.getElementById('colour-scheme-toggle-dark');
    if (colourSchemeToggleDark) {
        colourSchemeToggleDark.addEventListener('click', () => {
            onClickToggleButton()
        });
    }
});

function applyColourScheme(scheme) {
    const colourSchemeToggleLight = document.getElementById('colour-scheme-toggle-light');
    const colourSchemeToggleDark = document.getElementById('colour-scheme-toggle-dark');
    if (scheme === "dark") {
        document.documentElement.classList.remove('theme-light');
        document.documentElement.classList.add('theme-dark');
        if (colourSchemeToggleLight) {
            colourSchemeToggleLight.classList.remove('is-hidden');
            colourSchemeToggleDark.classList.add('is-hidden');
        }
    } else {
        document.documentElement.classList.remove('theme-dark');
        document.documentElement.classList.add('theme-light');
        if (colourSchemeToggleDark) {
            colourSchemeToggleDark.classList.remove('is-hidden');
            colourSchemeToggleLight.classList.add('is-hidden');
        }
    }
}
