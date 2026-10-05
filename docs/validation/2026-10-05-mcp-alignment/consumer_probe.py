"""Generate a service with installed MCP Hulk and validate it against this SDK."""
import argparse
import asyncio
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import tempfile
from datetime import timedelta

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client


async def main(args):
    sdk = pathlib.Path(__file__).resolve().parents[3]
    mcp = pathlib.Path(args.mcp_root).resolve()
    binary = mcp / "bin" / "mcp-server.exe"
    env = {**os.environ, "GOWORK": "off"}
    with tempfile.TemporaryDirectory(prefix="sdk-mcp-consumer-") as directory:
        area = pathlib.Path(directory)
        shutil.copytree(mcp / "templates", area / "templates")
        (area / "config.yaml").write_text(
            "mcp:\n  server:\n    transport: stdio\n    protocol: '2024-11-05'\n"
            "  registry:\n    storage_path: ./registry\n    auto_save: false\n", encoding="utf-8")
        params = StdioServerParameters(command=str(binary), cwd=str(area),
            env={"HULK_ENV": "audit", "HULK_MCP_SERVER_TRANSPORT": "stdio",
                 "HULK_MCP_REGISTRY_STORAGE_PATH": str(area / "registry")})
        async with stdio_client(params) as (read, write):
            async with ClientSession(read, write, read_timeout_seconds=timedelta(seconds=30)) as session:
                await session.initialize()
                result = await session.call_tool("generate_project", {
                    "name": "sdk-consumer", "stack": "go", "path": str(area)})
                assert not result.isError, result
                assert json.loads(result.content[0].text)["status"] == "created", result
        project = area / "sdk-consumer"
        commands = [
            ["go", "mod", "edit", "-replace", f"github.com/vertikon/sdk-hulk.vertikon.com.br={sdk}"],
            ["go", "mod", "tidy"],
            ["go", "build", "./..."],
            ["go", "test", "-json", "-count=1", "-timeout", "120s", "./..."],
        ]
        for command in commands:
            result = await asyncio.to_thread(subprocess.run, command, cwd=project,
                env=env, capture_output=True, text=True, timeout=180,
                creationflags=subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0)
            print(result.stdout, end="", flush=True)
            print(json.dumps({"command": command, "exit_code": result.returncode,
                              "stderr": result.stderr}), flush=True)
            assert result.returncode == 0, command
        print(json.dumps({"status": "pass", "sdk_root": str(sdk),
            "sdk_go_mod_sha256": hashlib.sha256((sdk / "go.mod").read_bytes()).hexdigest(),
            "mcp_binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
            "generated_go_mod": (project / "go.mod").read_text(),
            "scope": "local generated Go service with SDK replace; synthetic identity tests"}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("--mcp-root", required=True)
    asyncio.run(asyncio.wait_for(main(parser.parse_args()), timeout=600))
