import {useState, useEffect} from 'react';
import {EventsOn} from "../wailsjs/runtime/runtime";
import {
    GetStoredCredentialsStatus,
    SelectCredentialsFile,
    AuthenticateSource,
    AuthenticateTarget,
    ExportData,
    ImportData
} from "../wailsjs/go/main/App";

function App() {
    const [hasCredentials, setHasCredentials] = useState(false);
    const [sourceAuthed, setSourceAuthed] = useState(false);
    const [targetAuthed, setTargetAuthed] = useState(false);
    const [exportPath, setExportPath] = useState<string | null>(null);
    const [status, setStatus] = useState("initializing...");
    const [importing, setImporting] = useState(false);

    useEffect(() => {
        GetStoredCredentialsStatus().then(stored => {
            setHasCredentials(stored);
            setStatus(stored ? "Ready" : "GCP credentials required");
        }).catch(() => setStatus("error"));
    }, []);

    useEffect(() => {
        if (!importing) return;
        const offProgress = EventsOn("import:progress", (data: any) => {
            setStatus(`Importing ${data.category}: ${data.item}  (${data.current} / ${data.total})`);
        });
        const offDone = EventsOn("import:done", (msg: string) => {
            setImporting(false);
            setStatus("Import done: " + msg);
        });
        const offErr = EventsOn("import:error", (err: string) => {
            setImporting(false);
            setStatus("Import error: " + err);
        });
        return () => {
            offProgress();
            offDone();
            offErr();
        };
    }, [importing]);

    async function pickFile() {
        setStatus("selecting file...");
        try {
            const result = await SelectCredentialsFile();
            if (result === "saved") {
                setHasCredentials(true);
                setStatus("Credentials saved. Ready.");
            }
        } catch (e: any) {
            setStatus("error: " + e);
        }
    }

    async function loginSource() {
        setStatus("authenticating source...");
        try {
            const result = await AuthenticateSource();
            setSourceAuthed(result.includes("authenticated"));
            setStatus("Source: " + result);
        } catch (e: any) {
            setStatus("source error: " + e);
        }
    }

    async function loginTarget() {
        setStatus("authenticating target...");
        try {
            const result = await AuthenticateTarget();
            setTargetAuthed(result.includes("authenticated"));
            setStatus("Target: " + result);
        } catch (e: any) {
            setStatus("target error: " + e);
        }
    }

    async function doExport() {
        setStatus("exporting... this may take a while");
        try {
            const path = await ExportData();
            setExportPath(path);
            setStatus("Export saved to: " + path);
        } catch (e: any) {
            setStatus("export error: " + e);
        }
    }

    async function doImport() {
        if (!exportPath) {
            setStatus("no export data. run Export first.");
            return;
        }
        setImporting(true);
        setStatus("starting import...");
        try {
            const msg = await ImportData(exportPath);
            // msg == "import started" — real progress comes via events
        } catch (e: any) {
            setImporting(false);
            setStatus("import error: " + e);
        }
    }

    return (
        <div style={{padding: "40px", fontFamily: "system-ui, sans-serif"}}>
            <h1>ytmigrator</h1>
            <p style={{color: "#aab"}}>YouTube account data migration tool</p>

            <div style={{
                margin: "20px 0", padding: "15px",
                background: "#1e2a3a", borderRadius: "8px",
                fontFamily: "monospace", fontSize: "14px",
                color: "#d0d8e0", minHeight: "20px"
            }}>
                {importing && <span style={{color: "#4fc3f7"}}>⏳ </span>}
                {status}
            </div>

            {!hasCredentials && (
                <button onClick={pickFile} style={btnStyle("#1a73e8")}>
                    Select client_secret.json
                </button>
            )}

            {hasCredentials && (
                <div style={{display: "flex", gap: "10px", flexWrap: "wrap", marginBottom: "20px"}}>
                    <button onClick={loginSource} style={btnStyle(sourceAuthed ? "#2e7d32" : "#6aa84f")}>
                        {sourceAuthed ? "Re-auth Source" : "Login Source (Read)"}
                    </button>
                    <button onClick={loginTarget} style={btnStyle(targetAuthed ? "#c62828" : "#d9534f")}>
                        {targetAuthed ? "Re-auth Target" : "Login Target (Write)"}
                    </button>
                </div>
            )}

            {sourceAuthed && (
                <div style={{marginBottom: "10px"}}>
                    <button onClick={doExport} style={btnStyle("#f4b400")}>
                        Export Data
                    </button>
                </div>
            )}

            {targetAuthed && exportPath && !importing && (
                <div>
                    <button onClick={doImport} style={btnStyle("#7b1fa2")}>
                        Import Data to Target
                    </button>
                </div>
            )}

            {importing && (
                <div style={{marginTop: "10px", color: "#aab", fontSize: "13px"}}>
                    Import running in background. Progress auto-saves.<br/>
                    You may close the window and resume later.
                </div>
            )}
        </div>
    );
}

function btnStyle(bg: string) {
    return {
        padding: "12px 24px", fontSize: "15px",
        background: bg, color: "white",
        border: "none", borderRadius: "4px",
        cursor: "pointer", fontWeight: "bold"
    } as React.CSSProperties;
}

export default App;
