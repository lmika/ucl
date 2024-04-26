import "xterm";
import "wasm_exec";


var term = new Terminal({
    lineHeight: 1.2,
});
term.open(document.getElementById('terminal'));

function startSession(term) {
    let buffer = "";
    let lineBuffer = "";

    term.write('> ');

    term.onKey((ev, dom) => {
        const {key} = ev;

        switch (key) {
            case '\r':
                // Enter
                term.writeln('');

                let wantContinue = lineBuffer.length > 0;
                buffer += lineBuffer;
                lineBuffer = '';

                ucl.eval(buffer, wantContinue);
                break;
            case '\u007F':
                // Backspace
                if (lineBuffer.length > 0) {
                    term.write([0x08, 0x20, 0x08]);
                    lineBuffer = lineBuffer.slice(0, lineBuffer.length - 1);
                }
                break;
            default:
                if (key >= ' ') {
                    term.write(key);
                    lineBuffer += key;
                }
        }
    });

    ucl.onContinue = () => {
        buffer += "\n";
        lineBuffer = '';
        term.write(': ');
    }
    ucl.onNewCommand = () => {
        term.write('> ');
        buffer = '';
        lineBuffer = '';
    }
    ucl.onOutLine = (line) => { term.writeln(line); }
    ucl.onError = (err) => { term.writeln('error: ' + err); }
    term.focus();
}

const go = new Go();
WebAssembly.instantiateStreaming(fetch("/playwasm.wasm"), go.importObject)
    .then((result) => {
        go.run(result.instance);
        startSession(term);
    });