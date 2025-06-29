### `maogai-kang-mcqs-2023`

这是一个为了帮助同学们高效复习“毛概”选择题而制作的刷题库项目。

### 🚀 快速开始 (Quick Start)

以下步骤将指导您如何快速下载并运行本项目的预编译版本。

#### 1\. 前置要求 (Prerequisites)

  - 本项目的预编译版本原生支持 **Windows** 和 **Linux** 操作系统。
  - **macOS 用户**：此快速启动方法不直接支持 macOS。请参考相关开发文档进行源码编译部署，或在虚拟机中运行。

#### 2\. 启动项目 (Launch the Project)

**第一步：下载预编译程序 (Download the Executable)**

  - **对于 Windows 用户**:

      - **推荐**: 点击下方链接直接下载最新版本 (`v1.0.0`)。
          - [**maogai-kang-mcqs-2023-windows-with-assets.exe**](https://github.com/ShaddockNH3/maogai-kang-mcqs-2023/releases/download/v1.0.0/maogai-kang-mcqs-2023-windows-with-assets.exe)
      - 如果直连下载有问题，您也可以访问项目的 [**Releases 页面**](https://github.com/ShaddockNH3/maogai-kang-mcqs-2023/releases/tag/v1.0.0)，在页面底部的 **Assets** 部分手动下载 `maogai-kang-mcqs-2023-windows-with-assets.exe` 文件。

  - **对于 Linux 用户**:

      - 请访问项目的 [**Releases 页面**](https://github.com/ShaddockNH3/maogai-kang-mcqs-2023/releases/tag/v1.0.0)。
      - 在页面底部的 **Assets** 部分，下载 `maogai-kang-mcqs-2023-linux-with-assets` 文件。

> **注意**：无需下载Releases页面的Source code。

**第二步：运行程序 (Run the Program)**

  - **对于 Windows 用户**:
    直接双击下载的 `.exe` 文件即可启动。

  - **对于 Linux 用户**:
    首先需要为文件添加执行权限，然后在终端中运行它：

    ```bash
    # 假设文件已下载到当前目录
    chmod +x ./maogai-kang-mcqs-2023-linux-with-assets
    ./maogai-kang-mcqs-2023-linux-with-assets
    ```

> **重要提示**：程序启动后会占用一个终端窗口。**请勿关闭此终端窗口**，否则后台服务将会中断。

**第三步：在本机访问应用 (Access on Your Computer)**

1.  打开您的网页浏览器（如 Chrome, Edge, Firefox 等）。
2.  在地址栏输入并访问：`http://localhost:8899`
3.  程序启动成功！首次访问时，您可以随意创建一个用户名来开始使用。

> **请注意**: `localhost` 指的是“本机”，因此该地址**只能在运行程序的这台电脑上访问**。如需在手机等其他设备上访问，请参考下面的“本地网络访问”部分。

> **请注意**：如果端口号不为8899，则更改为对应的接口号进行访问

-----

### 📱 本地网络访问 (Local Network Access)

如果您想使用手机或其他设备（如平板）来刷题，只需进行以下设置：

1.  **确保网络连接**: 确保您运行程序的电脑和您的手机连接在 \*\*同一个局域网（Wi-Fi）\*\*下。

2.  **查找电脑的局域网IP地址**:

      - **Windows**: 打开“命令提示符(CMD)”或“PowerShell”，输入 `ipconfig` 命令，然后查找“无线局域网适配器 WLAN”或“以太网适配器”下的 “IPv4 地址”。它通常形如 `192.168.x.x`。
      - **Linux**: 在终端中输入 `ip addr` 或 `hostname -I` 命令来查找您的局域网IP地址。

3.  **在手机上访问**:

      - 打开手机浏览器，在地址栏输入 `http://<您查到的电脑IP地址>:8899` (请将 `<您查到的电脑IP地址>` 替换为您在第二步中找到的实际IP地址)。

> **防火墙提示**: 如果手机无法访问，请检查您电脑的防火墙设置，确保它允许其他设备访问您电脑的 `8899` 端口。您可能需要为该应用或端口添加入站规则（Inbound Rules）。

-----

### 🌐 公网服务器部署 (进阶)

如果您希望将此应用部署到云服务器上，让任何人都可以通过互联网访问，可以参考以下高级步骤。

**额外要求**:

  * 您需要拥有一台具有**公网 IP 地址**的云服务器 (VPS)，并已安装好 Linux 操作系统（如 Ubuntu, CentOS 等）。
  * 您已通过 `scp` 或其他方式将 **Linux 版本的可执行文件** 上传到服务器。

**部署步骤**:

1.  **添加执行权限**:
    通过 SSH 连接到您的服务器，并为程序添加执行权限。

    ```bash
    chmod +x ./maogai-kang-mcqs-2023-linux-with-assets
    ```

2.  **配置防火墙**:
    为了让外部用户能够访问应用，您需要开放应用所使用的端口（默认为 `8899`）。

      * **对于使用 UFW 的系统 (如 Ubuntu)**:
        ```bash
        sudo ufw allow 8899/tcp
        sudo ufw reload
        ```
      * **对于使用 firewalld 的系统 (如 CentOS)**:
        ```bash
        sudo firewall-cmd --zone=public --add-port=8899/tcp --permanent
        sudo firewall-cmd --reload
        ```

    > **云服务商提醒**: 除了服务器本身的防火墙，大部分云服务商（如阿里云、腾讯云、AWS）还有一层\*\*安全组（Security Group）\*\*防火墙。请确保您也在服务商的管理控制台中放行了 `8899` 端口的入站流量。

3.  **测试运行与访问**:

      * 先直接运行程序，检查它是否能正常启动：`./maogai-kang-mcqs-2023-linux-with-assets`
      * 此时，您应该可以在浏览器中通过 `http://<你的服务器公网IP>:8899` 访问您的应用了。确认可以访问后，在SSH终端按 `Ctrl + C` 停止它，准备下一步。

4.  **设置为后台服务 (推荐)**:
    为了让应用在您关闭 SSH 连接后依然能持续运行，并能开机自启，推荐使用 `systemd` 将其创建为一个服务。

    a.  创建一个 `systemd` 服务文件：`sudo nano /etc/systemd/system/maogai.service`

    b.  将以下内容粘贴到文件中。**请务必修改 `User`、`WorkingDirectory` 和 `ExecStart` 中的路径为您自己的实际情况**。
    \`\`\`ini
    [Unit]
    Description=Maogai Kang MCQs Service
    After=network.target

    ````
    [Service]
    Type=simple
    # 推荐使用一个非 root 用户运行，例如 'ubuntu' 或您自己的用户名
    User=your_username  
    # 程序所在的目录
    WorkingDirectory=/path/to/your/app/directory 
    # 程序的完整路径
    ExecStart=/path/to/your/app/directory/maogai-kang-mcqs-2023-linux-with-assets
    Restart=on-failure
    RestartSec=5s

    [Install]
    WantedBy=multi-user.target
    ```
    ````

    c.  保存并关闭文件后，运行以下命令来启用并启动服务：
    ` bash sudo systemctl daemon-reload  # 重新加载 systemd 配置 sudo systemctl enable maogai    # 设置开机自启 sudo systemctl start maogai     # 立即启动服务  `

    d.  您可以使用 `sudo systemctl status maogai` 来查看服务的运行状态。

**后续优化 (可选)**:

  * **域名解析**: 将您的域名指向服务器的公网 IP 地址。
  * **配置反向代理**: 使用 Nginx 或 Caddy 等 Web 服务器作为反向代理。这可以让您通过域名直接访问（无需端口号），并且能方便地配置 HTTPS 加密，让访问更安全。
