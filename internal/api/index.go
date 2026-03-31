package api

const indexHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Blockchain Node Dashboard</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { background-color: #f8f9fa; font-family: sans-serif; }
        .block-card { border-left: 5px solid #0d6efd; margin-bottom: 15px; }
        .hash-text { font-family: monospace; font-size: 0.8rem; word-break: break-all; color: #6c757d; }
        .pow-badge { background-color: #ffca28; color: #000; padding: 2px 8px; border-radius: 4px; font-size: 0.8rem; }
        .pos-badge { background-color: #4caf50; color: #fff; padding: 2px 8px; border-radius: 4px; font-size: 0.8rem; }
    </style>
</head>
<body>
    <nav class="navbar navbar-dark bg-dark mb-4">
        <div class="container">
            <span class="navbar-brand">⛓️ Go Hybrid Blockchain</span>
            <span class="badge bg-success">Node Online</span>
        </div>
    </nav>

    <div class="container">
        <div class="row mb-4">
            <div class="col-md-8">
                <button onclick="mine('pow')" class="btn btn-warning me-2" id="btnPow">⛏️ Mine PoW</button>
                <button onclick="mine('pos')" class="btn btn-success" id="btnPos">🛡️ Mint PoS</button>
            </div>
            <div class="col-md-4 text-end">
                <button onclick="loadChain()" class="btn btn-outline-primary">🔄 Refresh</button>
            </div>
        </div>

        <div id="miningStatus" class="alert alert-info d-none">Mining... Please wait</div>

        <div id="chainDisplay"></div>
    </div>

    <script>
        function loadChain() {
            fetch('/chain')
                .then(response => response.json())
                .then(blocks => {
                    const display = document.getElementById('chainDisplay');
                    display.innerHTML = '';
                    blocks.reverse().forEach(block => {
                        const isPoS = block.validator !== "";
                        const badge = isPoS ? '<span class="pos-badge">Proof-of-Stake</span>' : '<span class="pow-badge">Proof-of-Work</span>';
                        const extraInfo = isPoS ? '👤 Validator: <b>' + block.validator + '</b>' : '⚙️ Nonce: <b>' + block.nonce + '</b>';
                        
                        const html = '<div class="card block-card shadow-sm">' +
                            '<div class="card-body">' +
                                '<div class="d-flex justify-content-between"><h5>Block #' + block.height + '</h5>' + badge + '</div>' +
                                '<div class="row">' +
                                    '<div class="col-6 small"><strong>Hash:</strong><br><span class="hash-text">' + block.hash + '</span></div>' +
                                    '<div class="col-6 small"><strong>Prev:</strong><br><span class="hash-text">' + block.prev_block_hash + '</span></div>' +
                                '</div>' +
                                '<hr><div class="row small">' +
                                    '<div class="col-4">📅 ' + new Date(block.timestamp * 1000).toLocaleString() + '</div>' +
                                    '<div class="col-4 text-center">' + extraInfo + '</div>' +
                                    '<div class="col-4 text-end">TXs: ' + (block.transactions ? block.transactions.length : 0) + '</div>' +
                                '</div>' +
                            '</div>' +
                        '</div>';
                        display.innerHTML += html;
                    });
                });
        }

        function mine(type) {
            const status = document.getElementById('miningStatus');
            status.classList.remove('d-none');
            const url = type === 'pow' ? '/mine' : '/mine/pos';
            fetch(url).then(() => {
                status.classList.add('d-none');
                loadChain();
            });
        }

        loadChain();
        setInterval(loadChain, 5000);
    </script>
</body>
</html>
`