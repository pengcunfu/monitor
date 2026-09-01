#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
熔岩网络安全事件应急处置系统（monitor）一键部署脚本

按 go-web-deploy 技能规范实现，流程：
  1. 读取 scripts/.env（SSH 连接信息 + 敏感数据），真实环境变量优先级更高
  2. 本地交叉编译 Linux 二进制（入口 backend/cmd/server），产物输出到仓库根目录 dist/
  3. 动态生成运行时配置（jwt_secret 取自 .env，避免敏感值入库），上传远端
  4. SSH 连接 → 上传二进制与配置 → root+systemd 环境装为 systemd 服务，否则 nohup
  5. 校验进程与监听端口（含回环监听告警），输出访问地址 http://{host}:{port}/

仓库为 monorepo：Go 模块在 backend/（go.mod 位于 backend/），前端已构建并 embed。
配置约定：
  - 非敏感且与本仓库绑定的配置（应用名/端口/编译入口/仓库路径）写死在代码中
  - 环境相关的 SSH 连接信息（SSH_HOST/USER/PORT/ARCH）与敏感数据
    （DEPLOY_PASSWORD/JWT_SECRET）放 scripts/.env（已被 .gitignore 排除），
    缺失时交互输入

依赖：Python 3.8+，pip install paramiko
"""

import getpass
import os
import re
import subprocess
import sys
import time

try:
    import paramiko
except ImportError:
    sys.exit("缺少依赖 paramiko，请先安装：pip install paramiko")

try:
    sys.stdout.reconfigure(encoding="utf-8")  # Windows GBK 控制台避免中文乱码
except Exception:
    pass


def log(msg):
    print(msg, flush=True)


# ---------------------------------------------------------------------------
# 非敏感配置：与本仓库绑定的固定项（写死在代码中）
# ---------------------------------------------------------------------------
REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))  # 仓库根目录
BACKEND_DIR = os.path.join(REPO_ROOT, "backend")  # go.mod 所在目录（编译 cwd）
DIST_DIR = os.path.join(REPO_ROOT, "dist")  # 构建产物目录（根目录，已被 .gitignore 排除）
ENV_FILE = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env")

APP_NAME = "monitor"
APP_PORT = 8090  # 8080 被本机 nptunnel-server（内网穿透）占用，改用 8090
PKG = "./cmd/server"  # 编译入口包（相对 BACKEND_DIR）

# 远端运行时配置模板（jwt_secret 动态填充自 .env，避免敏感值硬编码入库）
CONF_TEMPLATE = """\
# {app} 运行时配置（由 scripts/deploy.py 自动生成并上传，勿手工编辑）
server:
  addr: ":{port}"
  db_path: "{conf_dir}/{app}.db"
  jwt_secret: "{jwt_secret}"
  jwt_expire_h: 72
collector:
  interval_sec: 10
  process_interval_sec: 30
  service_interval_sec: 30
  process_top_n: 20
"""


# ---------------------------------------------------------------------------
# 配置读取：scripts/.env（已 gitignore），真实环境变量优先，缺失时交互输入
# ---------------------------------------------------------------------------
def load_env(path):
    """解析 .env：KEY=value，支持 # 注释、export 前缀、单/双引号值。

    用 setdefault 写入 os.environ，保证真实环境变量优先于 .env。
    """
    if not os.path.exists(path):
        return
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if line.startswith("export "):
                line = line[len("export "):].strip()
            key, _, value = line.partition("=")
            key = key.strip()
            if not key:
                continue
            value = value.strip()
            if len(value) >= 2 and value[0] in "\"'" and value[-1] == value[0]:
                value = value[1:-1]
            os.environ.setdefault(key, value)


def env_or_input(key, prompt):
    """取普通配置；缺失时交互输入。"""
    v = os.environ.get(key)
    if not v:
        v = input(prompt).strip()
        os.environ[key] = v
    return v


def env_or_secret(key, prompt):
    """取敏感配置（密码/密钥）；缺失时交互输入。"""
    v = os.environ.get(key)
    if not v:
        v = getpass.getpass(prompt)
        os.environ[key] = v
    return v


def remote_paths(ssh):
    """根据 root/非 root 决定远端二进制与配置目录（不写死具体路径）。"""
    if is_root(ssh):
        bin_dir, conf_dir = "/usr/local/bin", f"/etc/{APP_NAME}"
    else:
        _, home = run_remote(ssh, "echo $HOME", quiet=True)
        home = home.strip() or "/root"
        bin_dir = conf_dir = f"{home}/{APP_NAME}"
    remote_bin = f"{bin_dir}/{APP_NAME}"
    remote_conf = f"{conf_dir}/deploy.yaml"
    return bin_dir, conf_dir, remote_bin, remote_conf


# ---------------------------------------------------------------------------
# 步骤 1：交叉编译 Linux 二进制（产物输出到根目录 dist/）
# ---------------------------------------------------------------------------
def build_binary(arch):
    log(f"==> 交叉编译 {PKG} (linux/{arch}) ...")
    os.makedirs(DIST_DIR, exist_ok=True)
    binary = os.path.join(DIST_DIR, f"{APP_NAME}-linux-{arch}")
    env = dict(os.environ, CGO_ENABLED="0", GOOS="linux", GOARCH=arch)
    cmd = ["go", "build", "-trimpath", "-ldflags=-s -w", "-o", binary, PKG]
    log("     " + " ".join(cmd))
    r = subprocess.run(cmd, env=env, cwd=BACKEND_DIR, capture_output=True, text=True)
    if r.returncode != 0:
        log("编译失败：")
        log(r.stderr or r.stdout or "（无输出）")
        sys.exit(1)
    size = os.path.getsize(binary) / 1024
    log(f"     完成，二进制大小 {size:.0f} KiB -> {binary}")
    return binary


# ---------------------------------------------------------------------------
# 步骤 2：SSH 连接与远程执行
# ---------------------------------------------------------------------------
def connect(host, port, user, password):
    log(f"==> 连接 {user}@{host}:{port} ...")
    ssh = paramiko.SSHClient()
    # 测试场景：自动接受未知主机密钥（生产环境请改用已知主机校验）
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(host, port=port, username=user, password=password, timeout=15)
    log("     连接成功")
    return ssh


def run_remote(ssh, cmd, quiet=False):
    """执行远程命令，返回 (exit_code, stdout)。"""
    _, stdout, stderr = ssh.exec_command(cmd, timeout=60)
    out = stdout.read().decode("utf-8", "replace").strip()
    err = stderr.read().decode("utf-8", "replace").strip()
    rc = stdout.channel.recv_exit_status()
    if not quiet:
        if out:
            log(out)
        if err and rc != 0:
            log("   [stderr] " + err)
    return rc, out


def is_root(ssh):
    rc, out = run_remote(ssh, "id -u", quiet=True)
    return rc == 0 and out.strip() == "0"


def has_systemd(ssh):
    rc, _ = run_remote(ssh, "command -v systemctl >/dev/null 2>&1", quiet=True)
    return rc == 0


def upload(ssh, local_path, remote_path):
    sftp = ssh.open_sftp()
    try:
        sftp.put(local_path, remote_path)
    finally:
        sftp.close()


# ---------------------------------------------------------------------------
# 步骤 3：启动（systemd / nohup 双分支，互相清理对方残留进程）
# ---------------------------------------------------------------------------
def start_systemd(ssh, remote_bin, conf_dir, app_args):
    log("==> 安装并启动 systemd 服务")
    # 清掉上次 nohup 残留进程，避免旧进程占端口
    run_remote(ssh, f"pkill -f '{remote_bin}' || true", quiet=True)
    unit = f"""[Unit]
Description={APP_NAME} web service
After=network.target

[Service]
Type=simple
WorkingDirectory={conf_dir}
ExecStart={remote_bin}{(' ' + app_args) if app_args else ''}
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
"""
    unit_path = f"/etc/systemd/system/{APP_NAME}.service"
    # 通过 stdin 写入 unit 文件，避免 shell 转义问题
    stdin, stdout, stderr = ssh.exec_command(f"cat > {unit_path}", timeout=15)
    stdin.write(unit)
    stdin.channel.shutdown_write()
    rc = stdout.channel.recv_exit_status()
    if rc != 0:
        log("   写入 unit 文件失败：" + stderr.read().decode("utf-8", "replace"))
        return False
    # enable 开机自启；restart 确保已运行的旧进程也加载新二进制/新配置
    run_remote(ssh, f"systemctl daemon-reload && systemctl enable {APP_NAME} && systemctl restart {APP_NAME}")
    time.sleep(2)
    return True


def start_nohup(ssh, remote_bin, conf_dir, app_args):
    log("==> 使用 nohup 后台启动（非 root / 无 systemd 环境）")
    # 清掉上次 systemd 残留服务，避免占端口（best-effort）
    run_remote(ssh, f"systemctl disable --now {APP_NAME}.service 2>/dev/null || true", quiet=True)
    run_remote(ssh, f"pkill -f '{remote_bin}' || true", quiet=True)
    time.sleep(1)
    script = (
        f"cd {conf_dir}\n"
        f"nohup {remote_bin}{(' ' + app_args) if app_args else ''} "
        f">> {conf_dir}/{APP_NAME}.log 2>&1 &"
    )
    run_remote(ssh, script)
    time.sleep(2)


def verify(ssh, remote_bin):
    log("==> 校验部署结果")
    _, out = run_remote(ssh, f"pgrep -af '{remote_bin}' || true", quiet=True)
    if out:
        log("   进程：\n   " + out.replace("\n", "\n   "))
    else:
        log("   [警告] 未发现进程！")
    _, out = run_remote(
        ssh,
        f"ss -tlnp 2>/dev/null | grep -E ':{APP_PORT}\\b' "
        f"|| netstat -tlnp 2>/dev/null | grep -E ':{APP_PORT}\\b' "
        f"|| echo NOT_FOUND",
        quiet=True,
    )
    if "NOT_FOUND" not in out:
        log(f"   端口 {APP_PORT} 正在监听：\n   " + out.replace("\n", "\n   "))
        # 回环告警：仅监听 127.0.0.1 时外网浏览器无法访问
        if re.search(rf"127\.0\.0\.1:{APP_PORT}\b", out) and not re.search(
            rf"(0\.0\.0\.0|\*|\[::\]):{APP_PORT}\b", out
        ):
            log(f"   [警告] 服务仅监听回环地址 127.0.0.1，外网浏览器无法访问；需让应用监听 0.0.0.0")
    else:
        log(f"   [警告] 未检测到端口 {APP_PORT} 监听（可能尚未就绪，请稍后检查日志）")


# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------
def main():
    log(f"==> 读取配置 {ENV_FILE}（真实环境变量优先）")
    load_env(ENV_FILE)

    # 环境相关连接信息与敏感数据：来自 .env（缺失时交互输入），不写死在代码中
    host = env_or_input("SSH_HOST", "远程主机 IP: ")
    user = env_or_input("SSH_USER", "SSH 用户名: ")
    port = int(env_or_input("SSH_PORT", "SSH 端口 [22]: ") or "22")
    arch = env_or_input("SSH_ARCH", "远程架构 [amd64/arm64]: ") or "amd64"
    if arch not in ("amd64", "arm64"):
        sys.exit(f"SSH_ARCH 仅支持 amd64/arm64，当前为：{arch}")
    password = env_or_secret("DEPLOY_PASSWORD", f"{user}@{host} 的密码: ")
    jwt_secret = env_or_secret("JWT_SECRET", "JWT 签名密钥: ")

    local_bin = build_binary(arch)

    ssh = connect(host, port, user, password)
    try:
        root = is_root(ssh)
        sysd = has_systemd(ssh)
        # 远端目录按 root/非 root 动态决定
        bin_dir, conf_dir, remote_bin, remote_conf = remote_paths(ssh)
        app_args = f"--config {remote_conf}"

        # 动态生成远端运行时配置（含敏感 jwt_secret，落在 gitignored 的 dist/ 下）
        os.makedirs(DIST_DIR, exist_ok=True)
        local_conf = os.path.join(DIST_DIR, "deploy.yaml")
        with open(local_conf, "w", encoding="utf-8") as f:
            f.write(CONF_TEMPLATE.format(app=APP_NAME, port=APP_PORT, conf_dir=conf_dir, jwt_secret=jwt_secret))
        log(f"==> 已生成运行时配置 {local_conf}（jwt_secret 来自 .env）")

        log("==> 上传二进制与配置")
        run_remote(ssh, f"mkdir -p {bin_dir} {conf_dir}")
        # 运行中的二进制不可直接覆写（ETXTBSY），先传临时文件再原子替换
        upload(ssh, local_bin, remote_bin + ".new")
        run_remote(ssh, f"mv -f {remote_bin}.new {remote_bin}")
        run_remote(ssh, f"chmod +x {remote_bin}")
        upload(ssh, local_conf, remote_conf)

        start_mode = "nohup"
        if root and sysd:
            if start_systemd(ssh, remote_bin, conf_dir, app_args):
                start_mode = "systemd"
            else:
                start_nohup(ssh, remote_bin, conf_dir, app_args)
        else:
            mode = "root 但无 systemd" if root else "非 root 用户"
            log(f"==> 环境：{mode}，改用 nohup 启动")
            start_nohup(ssh, remote_bin, conf_dir, app_args)

        verify(ssh, remote_bin)
    finally:
        ssh.close()

    log("\n========== 部署完成 ==========")
    log(f"  应用名     : {APP_NAME}")
    log(f"  访问地址   : http://{host}:{APP_PORT}/")
    log(f"  本地验证   : 浏览器打开上方地址，或 curl http://{host}:{APP_PORT}/")
    if start_mode == "systemd":
        log(f"  启动方式   : systemd（systemctl status {APP_NAME}、journalctl -u {APP_NAME} -e）")
    else:
        log(f"  启动方式   : nohup（日志 {conf_dir}/{APP_NAME}.log）")
    log(f"  配置文件   : {remote_conf}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
