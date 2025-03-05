// DataTables HTMX Integration

function dataTableConfig(tableID) {
    return {
        pageLength: 10,
        order: [], // Ordered based on what the server returns
        // Preserve HTMX functionality on pagination
        drawCallback: function (settings) {
            // Reinitialize HTMX on newly rendered rows
            const tableId = settings.nTable.id;
            const tableElement = document.querySelector(`#${tableId}`);
            if (tableElement) {
                htmx.process(tableElement);
            }
        },
        columnDefs: [
            {
                targets: '.numeric-column',
                className: 'narrow-numeric',
                width: 'auto',
            },
        ],
        layout: {
            topStart: 'search',
            topEnd: 'pageLength',
            bottomStart: 'info',
            bottomEnd: 'paging'
        },
    };
}

// DataTables HTMX Integration

/**
 * This script manages DataTables.net tables within an HTMX application.
 * It handles tables that dynamically appear and disappear from the DOM.
 */

// Wrap everything in a self-executing function that waits for DOM ready
(function () {
    // DOM ready function
    function ready(callback) {
        if (document.readyState !== 'loading') {
            // Document is already ready, call the callback directly
            callback();
        } else if (document.addEventListener) {
            // Modern browsers
            document.addEventListener('DOMContentLoaded', callback);
        } else {
            // IE <= 8
            document.attachEvent('onreadystatechange', function () {
                if (document.readyState === 'complete') {
                    callback();
                }
            });
        }
    }

    // Execute when DOM is ready
    ready(function () {
        // Store initialized DataTables instances
        const tableRegistry = new Map();

        /**
         * Initialize a DataTable instance
         * @param {HTMLElement} tableEl - The table element to initialize
         * @returns {object} - The DataTable instance
         */
        function initializeDataTable(tableEl) {
            const tableId = tableEl.id || `dt-${Math.random().toString(36).substr(2, 9)}`;

            // If table doesn't have an ID, assign one
            if (!tableEl.id) {
                tableEl.id = tableId;
            }

            // Get custom configuration from data attributes
            const configAttr = tableEl.getAttribute('data-dt-config');
            const config = configAttr ? JSON.parse(configAttr) : {};

            // Default configuration
            const defaultConfig = dataTableConfig(tableId);

            // Server-side configuration if data-server-side attribute exists
            if (tableEl.hasAttribute('data-server-side')) {
                const endpoint = tableEl.getAttribute('data-api-endpoint');
                if (endpoint) {
                    defaultConfig.serverSide = true;
                    defaultConfig.ajax = {
                        url: endpoint,
                        type: 'POST',
                        data: function (d) {
                            // Add HTMX request indicator
                            d.htmx = true;
                            return JSON.stringify(d);
                        },
                        contentType: 'application/json'
                    };
                }
            }

            // Merge configurations
            const mergedConfig = {...defaultConfig, ...config};

            // Initialize DataTable
            const dtInstance = $(tableEl).DataTable(mergedConfig);

            // Store reference to the instance
            tableRegistry.set(tableId, dtInstance);

            return dtInstance;
        }

        /**
         * Destroy a DataTable instance
         * @param {HTMLElement} tableEl - The table element to destroy
         */
        function destroyDataTable(tableEl) {
            if (!tableEl || !tableEl.id) return;

            const dtInstance = tableRegistry.get(tableEl.id);
            if (dtInstance) {
                dtInstance.destroy();
                tableRegistry.delete(tableEl.id);
            }
        }

        /**
         * Scan the container for DataTables and initialize them
         * @param {HTMLElement} container - The container to scan
         */
        function scanAndInitializeTables(container) {
            const tables = container.querySelectorAll('table.datatable:not(.initialized)');
            tables.forEach(table => {
                table.classList.add('initialized');
                initializeDataTable(table);
            });
        }

        /**
         * Clean up DataTables that are being removed
         * @param {HTMLElement} container - The container being removed/replaced
         */
        function cleanupRemovedTables(container) {
            const tables = container.querySelectorAll('table.datatable.initialized');
            tables.forEach(table => {
                destroyDataTable(table);
            });
        }

        // Listen for HTMX events
        document.body.addEventListener('htmx:afterSwap', function (event) {
            // Initialize DataTables in the new content
            scanAndInitializeTables(event.detail.target);
        });

        document.body.addEventListener('htmx:beforeSwap', function (event) {
            // Clean up any DataTables in the content being replaced
            cleanupRemovedTables(event.detail.target);
        });

        // Handle DataTables in loaded content
        document.body.addEventListener('htmx:load', function (event) {
            scanAndInitializeTables(document.body);
        });

        // Handle tables in HTMX-loaded content
        document.body.addEventListener('htmx:afterOnLoad', function (event) {
            scanAndInitializeTables(event.detail.target);
        });

        // Initialize tables on page load
        document.addEventListener('DOMContentLoaded', function () {
            scanAndInitializeTables(document.body);
        });

        // Export functions for direct use
        window.dtHtmx = {
            initializeDataTable,
            destroyDataTable,
            scanAndInitializeTables,
            getTableInstance: (tableId) => tableRegistry.get(tableId)
        };
    }); // End of ready() callback
})(); // End of self-executing function
