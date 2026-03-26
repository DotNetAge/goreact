document.addEventListener("DOMContentLoaded", function () {
    mermaid.initialize({
        startOnLoad: false,
        theme: "default",
        securityLevel: "loose",
        flowchart: {
            useMaxWidth: true,
            htmlLabels: true,
            curve: "basis"
        }
    });

    var graphs = document.querySelectorAll('pre.mermaid');
    graphs.forEach(function (graph, index) {
        graph.classList.add('mermaid-graph-' + index);
        mermaid.run('mermaid-graph-' + index);
    });
});