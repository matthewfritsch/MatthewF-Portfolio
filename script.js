// Theme toggle
const themeToggle = document.getElementById('theme-toggle');
const icon = themeToggle.querySelector('.icon');

const savedTheme = localStorage.getItem('theme') || 'dark';
if (savedTheme === 'light') {
    document.body.classList.add('light');
    icon.textContent = '☀️';
}

themeToggle.addEventListener('click', () => {
    document.body.classList.toggle('light');
    const isLight = document.body.classList.contains('light');
    icon.textContent = isLight ? '☀️' : '🌙';
    localStorage.setItem('theme', isLight ? 'light' : 'dark');
});

// Time display
function updateTime() {
    const date = new Date();
    const time = new Intl.DateTimeFormat('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        timeZone: 'America/Los_Angeles',
        hour12: true
    }).format(date);
    
    document.getElementById('time').textContent = time;
    
    const hour = new Date(date.toLocaleString('en-US', { timeZone: 'America/Los_Angeles' })).getHours();
    const responseTime = (hour > 20 || hour < 9) ? 'as soon as I can.' : 'shortly.';
    document.getElementById('response-time').textContent = responseTime;
}

updateTime();
setInterval(updateTime, 10000);

// Scroll reveal (progressive enhancement - content visible by default)
const revealSections = document.querySelectorAll('.about, .contact');
if (revealSections.length > 0 && 'IntersectionObserver' in window) {
    revealSections.forEach(el => el.classList.add('reveal'));

    const revealObserver = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('visible');
                revealObserver.unobserve(entry.target);
            }
        });
    }, { threshold: 0.1, rootMargin: '0px 0px -50px 0px' });

    revealSections.forEach(el => revealObserver.observe(el));
}
