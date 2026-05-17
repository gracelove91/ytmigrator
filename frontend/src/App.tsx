import {useState, useEffect} from 'react';
import {EventsOn} from "../wailsjs/runtime/runtime";
import {
    GetStoredCredentialsStatus,
    SelectCredentialsFile,
    AuthenticateSource,
    AuthenticateTarget,
    ExportData,
    GetExportInfo,
    ImportData,
} from "../wailsjs/go/main/App";
import {main} from "../wailsjs/go/models";

function App() {
    const [hasCredentials, setHasCredentials] = useState(false);
    const [sourceAuthed, setSourceAuthed] = useState(false);
    const [targetAuthed, setTargetAuthed] = useState(false);
    const [exportPath, setExportPath] = useState<string | null>(null);
    const [exportInfo, setExportInfo] = useState<any>(null);
    const [status, setStatus] = useState("initializing...");
    const [importing, setImporting] = useState(false);
    const [importOptions, setImportOptions] = useState({
        importSubscriptions: true,
        importPlaylists: true,
        importLikes: true,
    });

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
            const info = await GetExportInfo(path);
            setExportInfo(info);
            setStatus(`Export ready: ${info.subscriptionCount} subs, ${info.playlistCount} playlists, ${info.likedVideoCount} likes`);
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
        const opts = new main.ImportOptions({
            importSubscriptions: importOptions.importSubscriptions,
            importPlaylists: importOptions.importPlaylists,
            importLikes: importOptions.importLikes,
        });
        try {
            await ImportData(exportPath, opts);
        } catch (e: any) {
            setImporting(false);
            setStatus("import error: " + e);
        }
    }

    const anySelected = importOptions.importSubscriptions || importOptions.importPlaylists || importOptions.importLikes;

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

            {exportInfo && (
                <div style={{
                    margin: "20px 0", padding: "15px",
                    background: "#16202b", borderRadius: "8px",
                    color: "#d0d8e0", textAlign: "left"
                }}>
                    <h3 style={{margin: "0 0 10px 0", color: "#fff"}}>Export Preview</h3>
                    <div>📺 Subscriptions: {exportInfo.subscriptionCount}</div>
                    <div>📁 Playlists: {exportInfo.playlistCount} ({exportInfo.videoCount} videos)</div>
                    <div>❤️ Liked Videos: {exportInfo.likedVideoCount}</div>

                    <h4 style={{margin: "15px 0 8px 0", color: "#aab"}}>Select items to import:</h4>
                    <label style={{display: "block", margin: "5px 0", cursor: "pointer"}}>
                        <input type="checkbox" checked={importOptions.importSubscriptions}
                            onChange={(e) => setImportOptions({...importOptions, importSubscriptions: e.target.checked})}
                            style={{marginRight: "8px"}} />
                        Subscriptions ({exportInfo.subscriptionCount})
                    </label>
                    <label style={{display: "block", margin: "5px 0", cursor: "pointer"}}>
                        <input type="checkbox" checked={importOptions.importPlaylists}
                            onChange={(e) => setImportOptions({...importOptions, importPlaylists: e.target.checked})}
                            style={{marginRight: "8px"}} />
                        Playlists ({exportInfo.playlistCount})
                    </label>
                    <label style={{display: "block", margin: "5px 0", cursor: "pointer"}}>
                        <input type="checkbox" checked={importOptions.importLikes}
                            onChange={(e) => setImportOptions({...importOptions, importLikes: e.target.checked})}
                            style={{marginRight: "8px"}} />
                        Liked Videos ({exportInfo.likedVideoCount})
                    </label>

                    {targetAuthed && !importing && anySelected && (
                        <button onClick={doImport} style={{...btnStyle("#7b1fa2"), marginTop: "15px"}}>
                            Import Selected to Target
                        </button>
                    )}
                    {!targetAuthed && (
                        <div style={{marginTop: "10px", color: "#f4b400", fontSize: "13px"}}>
                            Login target account to enable import
                        </div>
                    )}
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
