import {useState, useEffect} from 'react';
import {GetStoredCredentialsStatus, SelectCredentialsFile, AuthenticateSource, AuthenticateTarget} from "../wailsjs/go/main/App";

function App() {
    const [hasCredentials, setHasCredentials] = useState<boolean>(false);
    const [status, setStatus] = useState<string>("initializing...");

    // 앱 시작 시 저장된 인증 정보 확인
    useEffect(() => {
        checkCredentials();
    }, []);

    async function checkCredentials() {
        try {
            const stored = await GetStoredCredentialsStatus();
            setHasCredentials(stored);
            setStatus(stored ? "GCP credentials loaded. Ready to authenticate." : "GCP credentials required.");
        } catch (e) {
            setStatus("error checking credentials");
        }
    }

    async function pickCredentialsFile() {
        setStatus("selecting file...");
        try {
            const result = await SelectCredentialsFile();
            setHasCredentials(true);
            setStatus(result === "saved" ? "Credentials saved successfully!" : "unexpected result");
        } catch (e: any) {
            setStatus("error: " + e);
        }
    }

    async function loginSource() {
        setStatus("authenticating source account... opening browser");
        try {
            const token = await AuthenticateSource();
            setStatus("Source token acquired: " + token.substring(0, 20) + "...");
        } catch (e: any) {
            setStatus("source auth error: " + e);
        }
    }

    async function loginTarget() {
        setStatus("authenticating target account... opening browser");
        try {
            const token = await AuthenticateTarget();
            setStatus("Target token acquired: " + token.substring(0, 20) + "...");
        } catch (e: any) {
            setStatus("target auth error: " + e);
        }
    }

    return (
        <div id="App" style={{padding: "40px", fontFamily: "sans-serif"}}>
            <h1>ytmigrator</h1>
            <p style={{color: "#666", marginBottom: "30px"}}>
                YouTube account data migration tool
            </p>

            <div style={{marginBottom: "20px", padding: "15px", background: "#f5f5f5", borderRadius: "8px"}}>
                <strong>Status:</strong> {status}
            </div>

            {!hasCredentials && (
                <div style={{marginBottom: "20px"}}>
                    <p style={{color: "#c00"}}>
                        First-time setup required. Download client_secret.json from Google Cloud Console.
                    </p>
                    <button
                        onClick={pickCredentialsFile}
                        style={{
                            padding: "12px 24px",
                            fontSize: "16px",
                            background: "#1a73e8",
                            color: "white",
                            border: "none",
                            borderRadius: "4px",
                            cursor: "pointer"
                        }}
                    >
                        Select client_secret.json
                    </button>
                </div>
            )}

            {hasCredentials && (
                <div style={{display: "flex", gap: "15px"}}>
                    <button
                        onClick={loginSource}
                        style={{
                            padding: "12px 24px",
                            fontSize: "16px",
                            background: "#34a853",
                            color: "white",
                            border: "none",
                            borderRadius: "4px",
                            cursor: "pointer"
                        }}
                    >
                        Login Source Account (Read)
                    </button>
                    <button
                        onClick={loginTarget}
                        style={{
                            padding: "12px 24px",
                            fontSize: "16px",
                            background: "#ea4335",
                            color: "white",
                            border: "none",
                            borderRadius: "4px",
                            cursor: "pointer"
                        }}
                    >
                        Login Target Account (Write)
                    </button>
                </div>
            )}
        </div>
    )
}

export default App;
