// NetCheck Dashboard JavaScript

document.addEventListener('DOMContentLoaded', function() {
    // Auto-refresh history every 10 seconds
    setInterval(function() {
        const historyEl = document.getElementById('history');
        if (historyEl && !document.querySelector('.htmx-request')) {
            htmx.trigger(historyEl, 'htmx:load');
        }
    }, 10000);

    // Form validation
    document.querySelectorAll('form[hx-post]').forEach(function(form) {
        form.addEventListener('submit', function(e) {
            const target = form.querySelector('[name="target"]');
            if (target && !target.value.trim()) {
                e.preventDefault();
                target.classList.add('is-invalid');
                return;
            }
            if (target) target.classList.remove('is-invalid');
        });
    });

    // Auto-dismiss alerts
    document.querySelectorAll('.alert-dismissible').forEach(function(alert) {
        setTimeout(function() {
            alert.style.transition = 'opacity 0.5s';
            alert.style.opacity = '0';
            setTimeout(function() {
                alert.remove();
            }, 500);
        }, 5000);
    });
});

// HTMX event handlers
document.body.addEventListener('htmx:beforeSwap', function(evt) {
    if (evt.detail.target.id === 'result') {
        // Scroll to result
        setTimeout(function() {
            const resultEl = document.getElementById('result');
            if (resultEl) {
                resultEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            }
        }, 100);
    }
});

document.body.addEventListener('htmx:afterRequest', function(evt) {
    if (evt.detail.failed) {
        const resultEl = document.getElementById('result');
        if (resultEl && evt.detail.target.id === 'result') {
            resultEl.innerHTML = '<div class="alert alert-danger">Check failed: ' +
                (evt.detail.xhr.responseText || 'Network error') + '</div>';
        }
    }
});
