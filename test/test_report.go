package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"
)

// HTMLReportData contains all data needed for the HTML report
type HTMLReportData struct {
	Summary     TestSummary
	GeneratedAt string
	Categories  []CategorySummary
}

// CategorySummary contains summary data for each test category
type CategorySummary struct {
	Name         string
	TotalTests   int
	PassedTests  int
	FailedTests  int
	PassRate     float64
	VariantStats []VariantCategoryStats
}

// VariantCategoryStats contains category statistics for a specific variant
type VariantCategoryStats struct {
	VariantName string
	Passed      int
	Failed      int
	Total       int
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Terminal Emulator Test Report</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        
        .header {
            background: rgba(255, 255, 255, 0.95);
            border-radius: 10px;
            padding: 30px;
            margin-bottom: 20px;
            box-shadow: 0 8px 32px rgba(31, 38, 135, 0.37);
            backdrop-filter: blur(4px);
            border: 1px solid rgba(255, 255, 255, 0.18);
        }
        
        .header h1 {
            color: #4a5568;
            margin-bottom: 10px;
            font-size: 2.5em;
            text-align: center;
        }
        
        .header .subtitle {
            text-align: center;
            color: #718096;
            font-size: 1.1em;
            margin-bottom: 20px;
        }
        
        .summary-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 20px;
            margin-top: 20px;
        }
        
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            transition: transform 0.3s ease;
        }
        
        .stat-card:hover {
            transform: translateY(-2px);
        }
        
        .stat-number {
            font-size: 2em;
            font-weight: bold;
            margin-bottom: 5px;
        }
        
        .stat-label {
            color: #718096;
            font-size: 0.9em;
        }
        
        .success { color: #38a169; }
        .failure { color: #e53e3e; }
        .warning { color: #d69e2e; }
        .info { color: #3182ce; }
        
        .content {
            background: rgba(255, 255, 255, 0.95);
            border-radius: 10px;
            padding: 30px;
            box-shadow: 0 8px 32px rgba(31, 38, 135, 0.37);
            backdrop-filter: blur(4px);
            border: 1px solid rgba(255, 255, 255, 0.18);
        }
        
        .tabs {
            display: flex;
            margin-bottom: 20px;
            background: #f8f9fa;
            border-radius: 8px;
            padding: 4px;
        }
        
        .tab {
            flex: 1;
            padding: 12px 20px;
            text-align: center;
            background: transparent;
            border: none;
            cursor: pointer;
            border-radius: 6px;
            transition: all 0.3s ease;
            font-weight: 500;
        }
        
        .tab.active, .tab:hover {
            background: white;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            color: #4a5568;
        }
        
        .tab-content {
            display: none;
        }
        
        .tab-content.active {
            display: block;
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        
        th, td {
            padding: 12px 15px;
            text-align: left;
            border-bottom: 1px solid #e2e8f0;
        }
        
        th {
            background: #4a5568;
            color: white;
            font-weight: 600;
            text-transform: uppercase;
            font-size: 0.85em;
            letter-spacing: 0.5px;
        }
        
        tr:hover {
            background: #f8f9fa;
        }
        
        .variant-header {
            background: #e2e8f0;
            font-weight: bold;
            color: #4a5568;
        }
        
        .test-passed {
            color: #38a169;
            font-weight: bold;
        }
        
        .test-failed {
            color: #e53e3e;
            font-weight: bold;
        }
        
        .test-id {
            font-family: monospace;
            background: #f1f5f9;
            padding: 2px 6px;
            border-radius: 4px;
            font-size: 0.9em;
        }
        
        .duration {
            font-family: monospace;
            color: #718096;
        }
        
        .error-details {
            background: #fed7d7;
            color: #c53030;
            padding: 8px;
            border-radius: 4px;
            font-size: 0.9em;
            margin-top: 5px;
            font-family: monospace;
        }

        .commands {
            font-family: monospace;
            font-size: 0.85em;
            max-width: 200px;
            word-wrap: break-word;
        }

        .commands code {
            background: #f7fafc;
            color: #2d3748;
            padding: 2px 4px;
            border-radius: 3px;
            border: 1px solid #e2e8f0;
            font-size: 0.9em;
        }
        
        .category-summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        
        .category-card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            transition: transform 0.3s ease;
        }
        
        .category-card:hover {
            transform: translateY(-2px);
        }
        
        .category-card h3 {
            color: #4a5568;
            margin-bottom: 15px;
            padding-bottom: 8px;
            border-bottom: 2px solid #e2e8f0;
        }
        
        .progress-bar {
            width: 100%;
            height: 8px;
            background: #e2e8f0;
            border-radius: 4px;
            overflow: hidden;
            margin: 10px 0;
        }
        
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #38a169, #4fd1c7);
            border-radius: 4px;
            transition: width 0.5s ease;
        }
        
        .variant-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-top: 15px;
        }
        
        .variant-stat {
            background: #f8f9fa;
            padding: 10px;
            border-radius: 6px;
            text-align: center;
            border-left: 4px solid #4a5568;
        }
        
        .filter-controls {
            margin: 20px 0;
            display: flex;
            gap: 15px;
            flex-wrap: wrap;
            align-items: center;
        }
        
        .filter-controls select, .filter-controls input {
            padding: 8px 12px;
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            background: white;
            color: #4a5568;
        }
        
        .export-btn {
            background: #4a5568;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-weight: 500;
            transition: background 0.3s ease;
        }
        
        .export-btn:hover {
            background: #2d3748;
        }
        
        .footer {
            text-align: center;
            margin-top: 30px;
            color: rgba(255, 255, 255, 0.8);
            font-size: 0.9em;
        }
        
        @media (max-width: 768px) {
            .container {
                padding: 10px;
            }
            
            .header {
                padding: 20px;
            }
            
            .header h1 {
                font-size: 2em;
            }
            
            .summary-stats {
                grid-template-columns: repeat(2, 1fr);
            }
            
            .filter-controls {
                flex-direction: column;
                align-items: stretch;
            }
            
            table {
                font-size: 0.9em;
            }
            
            th, td {
                padding: 8px;
            }
        }
        /* Minimal, flat, compact overrides */
        body { background: #ffffff !important; color: #111827 !important; line-height: 1.4 !important; }
        .container { max-width: 1100px !important; padding: 16px !important; }
        .header { background: transparent !important; border: none !important; box-shadow: none !important; border-bottom: 1px solid #e5e7eb !important; padding: 0 0 12px 0 !important; margin-bottom: 16px !important; }
        .header h1 { font-size: 22px !important; font-weight: 600 !important; color: #111827 !important; text-align: left !important; }
        .header .subtitle { margin-top: 4px !important; color: #6b7280 !important; font-size: 12px !important; text-align: left !important; }

        .content { background: transparent !important; border: none !important; box-shadow: none !important; padding: 8px 0 0 0 !important; }

        .summary-stats { gap: 8px !important; margin-top: 12px !important; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)) !important; }
        .stat-card { border: 1px solid #e5e7eb !important; background: #fff !important; padding: 10px !important; text-align: left !important; box-shadow: none !important; border-radius: 0 !important; }
        .stat-number { font-size: 18px !important; margin-bottom: 2px !important; color: #111827 !important; }
        .stat-label { color: #6b7280 !important; font-size: 12px !important; }
        .success { color: #059669 !important; }
        .failure { color: #dc2626 !important; }
        .info { color: #374151 !important; }

        .tabs { gap: 8px !important; background: transparent !important; padding: 0 !important; border-radius: 0 !important; border-bottom: 1px solid #e5e7eb !important; }
        .tab { background: none !important; border: none !important; box-shadow: none !important; border-radius: 0 !important; padding: 6px 8px !important; color: #374151 !important; font-size: 13px !important; }
        .tab.active { color: #111827 !important; border-bottom: 2px solid #111827 !important; }
        .tab:hover { color: #111827 !important; }

        table { margin: 12px 0 !important; border: 1px solid #e5e7eb !important; border-radius: 0 !important; box-shadow: none !important; }
        thead th { background: #f3f4f6 !important; color: #374151 !important; }
        th, td { padding: 6px 8px !important; border: 1px solid #e5e7eb !important; }
        tr:hover { background: transparent !important; }

        .test-id { background: #f9fafb !important; padding: 1px 4px !important; border-radius: 3px !important; font-size: 12px !important; }
        .duration { color: #6b7280 !important; }
        .error-details { background: #fef2f2 !important; color: #b91c1c !important; padding: 6px !important; border-radius: 3px !important; font-size: 12px !important; }

        .category-summary { gap: 8px !important; margin: 12px 0 !important; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)) !important; }
        .category-card { border: 1px solid #e5e7eb !important; background: #fff !important; padding: 10px !important; box-shadow: none !important; border-radius: 0 !important; }
        .category-card h3 { color: #111827 !important; margin-bottom: 8px !important; padding-bottom: 6px !important; border-bottom: 1px solid #e5e7eb !important; font-size: 14px !important; }
        .progress-bar { height: 6px !important; background: #e5e7eb !important; border-radius: 3px !important; margin: 8px 0 !important; }
        .progress-fill { background: #059669 !important; }
        .variant-grid { gap: 6px !important; margin-top: 8px !important; }
        .variant-stat { background: #f9fafb !important; padding: 8px !important; border-left: 3px solid #9ca3af !important; border-radius: 0 !important; }

        .filter-controls { margin: 10px 0 !important; gap: 8px !important; }
        .filter-controls select, .filter-controls input { padding: 6px 8px !important; border: 1px solid #e5e7eb !important; font-size: 12px !important; }
        .export-btn { background: #111827 !important; color: #fff !important; padding: 6px 10px !important; border-radius: 0 !important; box-shadow: none !important; }

        .footer { margin-top: 16px !important; color: #6b7280 !important; font-size: 12px !important; }

        @media (max-width: 768px) {
            .container { padding: 12px !important; }
            .header h1 { font-size: 18px !important; }
            .summary-stats { grid-template-columns: repeat(2, 1fr) !important; }
            table { font-size: 11px !important; }
            th, td { padding: 6px !important; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1> Terminal Emulator Test Report</h1>
            <div class="subtitle">Generated on {{.GeneratedAt}}</div>
            
            <div class="summary-stats">
                <div class="stat-card">
                    <div class="stat-number info">{{len .Summary.Variants}}</div>
                    <div class="stat-label">Variants Tested</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number info">{{.Summary.TotalTests}}</div>
                    <div class="stat-label">Total Tests</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number success">{{.Summary.TotalPassed}}</div>
                    <div class="stat-label">Passed</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number failure">{{.Summary.TotalFailed}}</div>
                    <div class="stat-label">Failed</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number info">{{printf "%.1f%%" .Summary.PassRate}}</div>
                    <div class="stat-label">Pass Rate</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number info">{{.Summary.TotalDuration}}</div>
                    <div class="stat-label">Duration</div>
                </div>
            </div>
        </div>
        
        <div class="content">
            <div class="tabs">
                <button class="tab active" onclick="showTab('overview')"> Overview</button>
                <button class="tab" onclick="showTab('detailed')"> Detailed Results</button>
                <button class="tab" onclick="showTab('categories')">📁 By Category</button>
                <button class="tab" onclick="showTab('failures')"> Failures</button>
            </div>
            
            <div id="overview" class="tab-content active">
                <h2>Variant Overview</h2>
                <table>
                    <thead>
                        <tr>
                            <th>Variant Name</th>
                            <th>Build Status</th>
                            <th>Total Tests</th>
                            <th>Passed</th>
                            <th>Failed</th>
                            <th>Pass Rate</th>
                            <th>Duration</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .Summary.Variants}}
                        <tr>
                            <td><strong>{{.Name}}</strong></td>
                            <td>{{if .BuildSuccess}}[OK] Success{{else}} Failed{{end}}</td>
                            <td>{{.TotalTests}}</td>
                            <td class="test-passed">{{.PassedTests}}</td>
                            <td class="test-failed">{{.FailedTests}}</td>
                            <td>{{printf "%.1f%%" .PassRate}}</td>
                            <td class="duration">{{.TotalDuration}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
            
            <div id="detailed" class="tab-content">
                <h2>Detailed Test Results</h2>
                <div class="filter-controls">
                    <select id="variantFilter" onchange="filterResults()">
                        <option value="">All Variants</option>
                        {{range .Summary.Variants}}
                        <option value="{{.Name}}">{{.Name}}</option>
                        {{end}}
                    </select>
                    <select id="statusFilter" onchange="filterResults()">
                        <option value="">All Results</option>
                        <option value="passed">Passed Only</option>
                        <option value="failed">Failed Only</option>
                    </select>
                    <input type="text" id="searchFilter" placeholder="Search test description..." oninput="filterResults()">
                    <button class="export-btn" onclick="exportToCSV()">Export to CSV</button>
                </div>
                
                <table id="resultsTable">
                    <thead>
                        <tr>
                            <th>Test ID</th>
                            <th>Category</th>
                            <th>Description</th>
                            <th>Commands</th>
                            <th>Variant</th>
                            <th>Status</th>
                            <th>Duration</th>
                            <th>Error</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .Summary.Variants}}
                            {{range .TestResults}}
                            <tr class="test-row" data-variant="{{.Variant}}" data-status="{{if .Passed}}passed{{else}}failed{{end}}" data-description="{{.TestCase.Description}}">
                                <td class="test-id">{{.TestCase.ID}}</td>
                                <td>{{.TestCase.Category}}</td>
                                <td>{{.TestCase.Description}}</td>
                                <td class="commands">
                                    {{range $i, $cmd := .TestCase.Commands}}
                                        {{if $i}}<br>{{end}}<code>{{$cmd}}</code>
                                    {{end}}
                                </td>
                                <td><strong>{{.Variant}}</strong></td>
                                <td class="{{if .Passed}}test-passed{{else}}test-failed{{end}}">
                                    {{if .Passed}}[PASS]{{else}}[FAIL]{{end}}
                                </td>
                                <td class="duration">{{.Duration}}</td>
                                <td>
                                    {{if .Error}}
                                        <div class="error-details">{{.Error}}</div>
                                    {{end}}
                                </td>
                            </tr>
                            {{end}}
                        {{end}}
                    </tbody>
                </table>
            </div>
            
            <div id="categories" class="tab-content">
                <h2>Results by Category</h2>
                <div class="category-summary">
                    {{range .Categories}}
                    <div class="category-card">
                        <h3>{{.Name}}</h3>
                        <div>
                            <strong>{{.PassedTests}}/{{.TotalTests}}</strong> tests passed
                            ({{printf "%.1f%%" .PassRate}})
                        </div>
                        <div class="progress-bar">
                            <div class="progress-fill" style="width: {{.PassRate}}%"></div>
                        </div>
                        
                        <div class="variant-grid">
                            {{range .VariantStats}}
                            <div class="variant-stat">
                                <div><strong>{{.VariantName}}</strong></div>
                                <div>{{.Passed}}/{{.Total}} passed</div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            
            <div id="failures" class="tab-content">
                <h2>Failed Tests Analysis</h2>
                {{$hasFailures := false}}
                {{range .Summary.Variants}}
                    {{$variant := .}}
                    {{range .TestResults}}
                        {{if not .Passed}}
                            {{if not $hasFailures}}
                                {{$hasFailures = true}}
                                <table>
                                    <thead>
                                        <tr>
                                            <th>Test ID</th>
                                            <th>Category</th>
                                            <th>Description</th>
                                            <th>Commands</th>
                                            <th>Variant</th>
                                            <th>Error Details</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                            {{end}}
                            <tr>
                                <td class="test-id">{{.TestCase.ID}}</td>
                                <td>{{.TestCase.Category}}</td>
                                <td>{{.TestCase.Description}}</td>
                                <td class="commands">
                                    {{range $i, $cmd := .TestCase.Commands}}
                                        {{if $i}}<br>{{end}}<code>{{$cmd}}</code>
                                    {{end}}
                                </td>
                                <td><strong>{{.Variant}}</strong></td>
                                <td>
                                    {{if .Error}}
                                        <div class="error-details">{{.Error}}</div>
                                    {{else}}
                                        <em>No error details available</em>
                                    {{end}}
                                </td>
                            </tr>
                        {{end}}
                    {{end}}
                {{end}}
                
                {{if $hasFailures}}
                    </tbody>
                </table>
                {{else}}
                    <div style="text-align: center; padding: 40px; color: #38a169;">
                        <h3>🎉 No Failed Tests!</h3>
                        <p>All tests passed successfully across all variants.</p>
                    </div>
                {{end}}
            </div>
        </div>
        
        <div class="footer">
            <p>Terminal Emulator Test Suite | Automated Testing Framework</p>
            <p>Report generated with ❤️ by Go Test Runner</p>
        </div>
    </div>
    
    <script>
        function showTab(tabName) {
            // Hide all tab contents
            const contents = document.querySelectorAll('.tab-content');
            contents.forEach(content => content.classList.remove('active'));
            
            // Remove active class from all tabs
            const tabs = document.querySelectorAll('.tab');
            tabs.forEach(tab => tab.classList.remove('active'));
            
            // Show selected tab content
            document.getElementById(tabName).classList.add('active');
            
            // Add active class to clicked tab
            event.target.classList.add('active');
        }
        
        function filterResults() {
            const variantFilter = document.getElementById('variantFilter').value;
            const statusFilter = document.getElementById('statusFilter').value;
            const searchFilter = document.getElementById('searchFilter').value.toLowerCase();
            
            const rows = document.querySelectorAll('.test-row');
            
            rows.forEach(row => {
                const variant = row.getAttribute('data-variant');
                const status = row.getAttribute('data-status');
                const description = row.getAttribute('data-description').toLowerCase();
                
                let show = true;
                
                if (variantFilter && variant !== variantFilter) show = false;
                if (statusFilter && status !== statusFilter) show = false;
                if (searchFilter && !description.includes(searchFilter)) show = false;
                
                row.style.display = show ? '' : 'none';
            });
        }
        
        function exportToCSV() {
            const table = document.getElementById('resultsTable');
            let csv = '';
            
            // Header
            const headers = Array.from(table.querySelectorAll('thead th')).map(th => th.textContent);
            csv += headers.join(',') + '\\n';
            
            // Rows
            const rows = table.querySelectorAll('tbody tr');
            rows.forEach(row => {
                if (row.style.display !== 'none') {
                    const cells = Array.from(row.querySelectorAll('td')).map(td => {
                        let text = td.textContent.trim().replace(/"/g, '""');
                        if (text.includes(',') || text.includes('\\n')) {
                            text = '"' + text + '"';
                        }
                        return text;
                    });
                    csv += cells.join(',') + '\\n';
                }
            });
            
            // Download
            const blob = new Blob([csv], { type: 'text/csv' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'terminal_emulator_test_results.csv';
            a.click();
            window.URL.revokeObjectURL(url);
        }
        
        // Initialize page
        document.addEventListener('DOMContentLoaded', function() {
            // Add some initial animations
            const statCards = document.querySelectorAll('.stat-card');
            statCards.forEach((card, index) => {
                setTimeout(() => {
                    card.style.transform = 'translateY(-5px)';
                    setTimeout(() => {
                        card.style.transform = 'translateY(0)';
                    }, 200);
                }, index * 100);
            });
        });

        // Minimal content sanitization and labels normalization
        document.addEventListener('DOMContentLoaded', function() {
            // Header title
            const h1 = document.querySelector('.header h1');
            if (h1) h1.textContent = 'Terminal Emulator Test Report';

            // Tabs labels
            const tabs = document.querySelectorAll('.tabs .tab');
            if (tabs.length >= 4) {
                tabs[0].textContent = 'Overview';
                tabs[1].textContent = 'Detailed Results';
                tabs[2].textContent = 'By Category';
                tabs[3].textContent = 'Failures';
            }

            // Overview build status cells -> plain Success/Failed
            document.querySelectorAll('#overview tbody tr td:nth-child(2)')
                .forEach(td => {
                    const t = (td.textContent || '').toLowerCase();
                    td.textContent = t.includes('fail') ? 'Failed' : 'Success';
                });

            // Detailed results status cells -> Pass/Fail based on class
            document.querySelectorAll('#detailed tbody tr td:nth-child(5)')
                .forEach(td => {
                    if (td.classList.contains('test-passed')) td.textContent = 'Pass';
                    else if (td.classList.contains('test-failed')) td.textContent = 'Fail';
                });

            // No failures banner title cleanup
            const nf = document.querySelector('#failures h3');
            if (nf) nf.textContent = 'No Failed Tests';

            // Footer cleanup
            const footerPs = document.querySelectorAll('.footer p');
            if (footerPs.length > 1) {
                footerPs[1].textContent = 'Report generated by Go Test Runner';
            }
        });
    </script>
</body>
</html>`

// GenerateHTMLReport generates a comprehensive HTML test report
func GenerateHTMLReport(summary TestSummary, outputPath string) error {
	// Ensure the reports directory exists
	reportsDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %v", err)
	}

	// Calculate additional metrics
	data := prepareReportData(summary)

	// Parse and execute template
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"printf": fmt.Sprintf,
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %v", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

// prepareReportData prepares the data structure for the HTML report
func prepareReportData(summary TestSummary) HTMLReportData {
	// Calculate pass rates
	for i := range summary.Variants {
		if summary.Variants[i].TotalTests > 0 {
			summary.Variants[i].PassRate = float64(summary.Variants[i].PassedTests) / float64(summary.Variants[i].TotalTests) * 100
		}
	}

	// Calculate overall pass rate
	if summary.TotalTests > 0 {
		summary.PassRate = float64(summary.TotalPassed) / float64(summary.TotalTests) * 100
	}

	// Prepare category summaries
	categories := prepareCategorySummaries(summary)

	return HTMLReportData{
		Summary:     summary,
		GeneratedAt: time.Now().Format("January 2, 2006 at 15:04:05 MST"),
		Categories:  categories,
	}
}

// prepareCategorySummaries creates category-wise summaries
func prepareCategorySummaries(summary TestSummary) []CategorySummary {
	categoryMap := make(map[string]*CategorySummary)

	// Collect stats for each category
	for _, variant := range summary.Variants {
		for _, result := range variant.TestResults {
			category := result.TestCase.Category
			
			if categoryMap[category] == nil {
				categoryMap[category] = &CategorySummary{
					Name:         category,
					VariantStats: []VariantCategoryStats{},
				}
			}

			cat := categoryMap[category]
			cat.TotalTests++
			if result.Passed {
				cat.PassedTests++
			} else {
				cat.FailedTests++
			}

			// Update variant stats for this category
			found := false
			for i := range cat.VariantStats {
				if cat.VariantStats[i].VariantName == variant.Name {
					cat.VariantStats[i].Total++
					if result.Passed {
						cat.VariantStats[i].Passed++
					} else {
						cat.VariantStats[i].Failed++
					}
					found = true
					break
				}
			}

			if !found {
				stat := VariantCategoryStats{
					VariantName: variant.Name,
					Total:       1,
				}
				if result.Passed {
					stat.Passed = 1
				} else {
					stat.Failed = 1
				}
				cat.VariantStats = append(cat.VariantStats, stat)
			}
		}
	}

	// Convert map to slice and calculate pass rates
	categories := make([]CategorySummary, 0, len(categoryMap))
	for _, cat := range categoryMap {
		if cat.TotalTests > 0 {
			cat.PassRate = float64(cat.PassedTests) / float64(cat.TotalTests) * 100
		}
		categories = append(categories, *cat)
	}

	return categories
}

