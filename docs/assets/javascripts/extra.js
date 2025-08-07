// Extra JavaScript for Nephoran Intent Operator documentation

// Clipboard icon SVG constant
const CLIPBOARD_SVG = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M19,21H8V7H19M19,5H8A2,2 0 0,0 6,7V21A2,2 0 0,0 8,23H19A2,2 0 0,0 21,21V7A2,2 0 0,0 19,5M16,1H4A2,2 0 0,0 2,3V17H4V3H16V1Z"/></svg>';

document.addEventListener('DOMContentLoaded', function() {
    
    // Add copy buttons to code blocks
    addCopyButtons();
    
    // Initialize tooltips for technical terms
    initializeTooltips();
    
    // Add scroll-to-top functionality
    addScrollToTop();
    
    // Initialize intent processing demo
    initializeIntentDemo();
    
    // Add theme toggle animation
    enhanceThemeToggle();
    
    // Initialize performance metrics animation
    animateMetrics();
});

/**
 * Add copy buttons to all code blocks
 */
function addCopyButtons() {
    const codeBlocks = document.querySelectorAll('pre > code');
    
    codeBlocks.forEach(function(codeBlock) {
        const pre = codeBlock.parentNode;
        const button = document.createElement('button');
        
        button.className = 'md-clipboard md-icon';
        button.title = 'Copy to clipboard';
        button.innerHTML = CLIPBOARD_SVG;
        
        button.addEventListener('click', function() {
            const text = codeBlock.textContent;
            
            navigator.clipboard.writeText(text).then(function() {
                button.classList.add('md-clipboard--copied');
                button.title = 'Copied!';
                
                setTimeout(function() {
                    button.classList.remove('md-clipboard--copied');
                    button.title = 'Copy to clipboard';
                }, 2000);
            }).catch(function(err) {
                console.error('Failed to copy text: ', err);
            });
        });
        
        pre.style.position = 'relative';
        pre.appendChild(button);
    });
}

/**
 * Initialize tooltips for technical terms
 */
function initializeTooltips() {
    const tooltipTerms = {
        'O-RAN': 'Open Radio Access Network - An open standard for RAN architecture',
        'AMF': 'Access and Mobility Management Function - Core network function for access and mobility',
        'SMF': 'Session Management Function - Manages PDU sessions in 5G core',
        'UPF': 'User Plane Function - Handles user data forwarding in 5G core',
        'NSSF': 'Network Slice Selection Function - Selects appropriate network slice',
        'RAG': 'Retrieval-Augmented Generation - AI technique combining retrieval with generation',
        'CRD': 'Custom Resource Definition - Kubernetes API extension mechanism',
        'GitOps': 'Operations model using Git as single source of truth'
    };
    
    Object.keys(tooltipTerms).forEach(function(term) {
        const regex = new RegExp(`\\b${term}\\b`, 'gi');
        const walker = document.createTreeWalker(
            document.body,
            NodeFilter.SHOW_TEXT,
            null,
            false
        );
        
        let textNode;
        const textNodes = [];
        
        while (textNode = walker.nextNode()) {
            if (textNode.parentNode.tagName !== 'CODE' && 
                textNode.parentNode.tagName !== 'PRE' &&
                !textNode.parentNode.classList.contains('md-tooltip')) {
                textNodes.push(textNode);
            }
        }
        
        textNodes.forEach(function(node) {
            if (regex.test(node.textContent)) {
                const parent = node.parentNode;
                const wrapper = document.createElement('span');
                wrapper.innerHTML = node.textContent.replace(regex, 
                    `<span class="md-tooltip" title="${tooltipTerms[term]}">${term}</span>`
                );
                
                parent.insertBefore(wrapper, node);
                parent.removeChild(node);
            }
        });
    });
}

/**
 * Add scroll-to-top button
 */
function addScrollToTop() {
    const button = document.createElement('button');
    button.className = 'scroll-to-top';
    button.innerHTML = '↑';
    button.title = 'Scroll to top';
    button.style.cssText = `
        position: fixed;
        bottom: 2rem;
        right: 2rem;
        width: 3rem;
        height: 3rem;
        border-radius: 50%;
        border: none;
        background: var(--md-primary-fg-color);
        color: var(--md-primary-bg-color);
        font-size: 1.5rem;
        cursor: pointer;
        opacity: 0;
        visibility: hidden;
        transition: all 0.3s ease;
        z-index: 100;
    `;
    
    document.body.appendChild(button);
    
    button.addEventListener('click', function() {
        window.scrollTo({ top: 0, behavior: 'smooth' });
    });
    
    window.addEventListener('scroll', function() {
        if (window.pageYOffset > 300) {
            button.style.opacity = '1';
            button.style.visibility = 'visible';
        } else {
            button.style.opacity = '0';
            button.style.visibility = 'hidden';
        }
    });
}

/**
 * Initialize intent processing demo
 */
function initializeIntentDemo() {
    const demoContainer = document.querySelector('.intent-demo');
    if (!demoContainer) return;
    
    const examples = [
        'Deploy a high-availability AMF instance for production',
        'Create a URLLC network slice for industrial IoT',
        'Set up O-DU with massive MIMO support',
        'Deploy complete 5G core with auto-scaling'
    ];
    
    let currentExample = 0;
    const textElement = demoContainer.querySelector('.demo-text');
    
    if (textElement) {
        function typeWriter(text, callback) {
            let i = 0;
            textElement.textContent = '';
            
            function type() {
                if (i < text.length) {
                    textElement.textContent += text.charAt(i);
                    i++;
                    setTimeout(type, 50);
                } else if (callback) {
                    setTimeout(callback, 2000);
                }
            }
            type();
        }
        
        function nextExample() {
            typeWriter(examples[currentExample], function() {
                currentExample = (currentExample + 1) % examples.length;
                setTimeout(nextExample, 1000);
            });
        }
        
        nextExample();
    }
}

/**
 * Enhance theme toggle with animation
 */
function enhanceThemeToggle() {
    const themeToggle = document.querySelector('[data-md-component="palette"]');
    if (!themeToggle) return;
    
    const toggleInput = themeToggle.querySelector('input[type="radio"]');
    if (!toggleInput) return;
    
    toggleInput.addEventListener('change', function() {
        document.body.style.transition = 'background-color 0.3s ease, color 0.3s ease';
        
        setTimeout(function() {
            document.body.style.transition = '';
        }, 300);
    });
}

/**
 * Animate metrics counters
 */
function animateMetrics() {
    const metricElements = document.querySelectorAll('.metric-value');
    
    const observer = new IntersectionObserver(function(entries) {
        entries.forEach(function(entry) {
            if (entry.isIntersecting) {
                const element = entry.target;
                const finalValue = parseInt(element.textContent.replace(/[^\d]/g, ''));
                const suffix = element.textContent.replace(/[\d]/g, '');
                
                animateCounter(element, 0, finalValue, 2000, suffix);
                observer.unobserve(element);
            }
        });
    });
    
    metricElements.forEach(function(element) {
        observer.observe(element);
    });
}

/**
 * Animate counter from start to end value
 */
function animateCounter(element, start, end, duration, suffix) {
    const range = end - start;
    const increment = range / (duration / 16);
    let current = start;
    
    function updateCounter() {
        current += increment;
        if (current >= end) {
            element.textContent = end + suffix;
        } else {
            element.textContent = Math.floor(current) + suffix;
            requestAnimationFrame(updateCounter);
        }
    }
    
    updateCounter();
}

/**
 * Add keyboard shortcuts
 */
document.addEventListener('keydown', function(event) {
    // Ctrl/Cmd + K for search
    if ((event.ctrlKey || event.metaKey) && event.key === 'k') {
        event.preventDefault();
        const searchInput = document.querySelector('[data-md-component="search-query"]');
        if (searchInput) {
            searchInput.focus();
        }
    }
    
    // Escape to close search
    if (event.key === 'Escape') {
        const searchInput = document.querySelector('[data-md-component="search-query"]');
        if (searchInput && document.activeElement === searchInput) {
            searchInput.blur();
        }
    }
});

/**
 * Initialize table sorting for API reference tables
 */
function initializeTableSorting() {
    const tables = document.querySelectorAll('.api-table');
    
    tables.forEach(function(table) {
        const headers = table.querySelectorAll('th');
        
        headers.forEach(function(header, index) {
            header.style.cursor = 'pointer';
            header.addEventListener('click', function() {
                sortTable(table, index);
            });
        });
    });
}

function sortTable(table, columnIndex) {
    const rows = Array.from(table.querySelectorAll('tr')).slice(1); // Skip header
    const isNumeric = rows.every(row => {
        const cell = row.cells[columnIndex];
        return cell && !isNaN(parseFloat(cell.textContent.trim()));
    });
    
    rows.sort(function(a, b) {
        const aVal = a.cells[columnIndex].textContent.trim();
        const bVal = b.cells[columnIndex].textContent.trim();
        
        if (isNumeric) {
            return parseFloat(aVal) - parseFloat(bVal);
        } else {
            return aVal.localeCompare(bVal);
        }
    });
    
    const tbody = table.querySelector('tbody') || table;
    rows.forEach(function(row) {
        tbody.appendChild(row);
    });
}

// Initialize table sorting when DOM is loaded
document.addEventListener('DOMContentLoaded', initializeTableSorting);

/**
 * Add external link indicators
 */
function addExternalLinkIndicators() {
    const externalLinks = document.querySelectorAll('a[href^="http"]');
    
    externalLinks.forEach(function(link) {
        if (!link.hostname.includes(window.location.hostname)) {
            link.classList.add('external-link');
            link.setAttribute('target', '_blank');
            link.setAttribute('rel', 'noopener noreferrer');
            
            // Add external link icon
            const icon = document.createElement('span');
            icon.innerHTML = ' ↗';
            icon.style.fontSize = '0.8em';
            icon.style.opacity = '0.7';
            link.appendChild(icon);
        }
    });
}

// Initialize external link indicators
document.addEventListener('DOMContentLoaded', addExternalLinkIndicators);