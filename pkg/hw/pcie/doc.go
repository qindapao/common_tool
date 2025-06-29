/*
PCIE AER（Advanced Error Reporting）使用说明

一、支持条件
 1. 硬件支持 PCIe AER capability。  
    执行：
      $ lspci -vv -s <DOMAIN:BUS:DEV.FUNC>
    在输出中找到：
      Advanced Error Reporting:
         UESta: …
         UEMsk: …
         CESta: …
         CEMsk: …
    如果看到“Advanced Error Reporting”段，即设备支持 AER。

 2. 内核启用了 AER 支持。  
    检查内核配置：
      $ grep CONFIG_PCIEAER /boot/config-$(uname -r)
      CONFIG_PCIEAER=y
    如未编译，可在启动参数中去除“pci=noaer”，并加载模块：
      $ modprobe aer

二、查看当前 AER 状态
 通过 lspci -vv 输出的 AER 段即可看到各寄存器：
    UESta  – Uncorrectable Error Status
    UEMsk  – Uncorrectable Error Mask
    UESvrt – Uncorrectable Error Severity
    CESta  – Correctable Error Status
    CEMsk  – Correctable Error Mask
    CESvrt – Correctable Error Severity

示例：
  $ lspci -vv -s 0000:00:1f.6
    …
    Advanced Error Reporting:
        UESta:  0000 CE   UEMsk: 000F    UESvrt: 0001
        CESta:  0000       CEMsk: 0001    CESvrt: 0000

三、手动开/关 AER：  
 1. 确定 PCI Express Capability 偏移  
    $ EXP=$(lspci -xxx -s 0000:00:1f.6 | grep -m1 "Exp Cap" | awk '{print $3}')
    （EXP 就是 “Exp Cap” 这一行的起始偏移，比如 0x50）

 2. 读写寄存器：  
    # 读 Uncorr Mask（2 字节）
    $ setpci -s 0000:00:1f.6 ${EXP}+04.w
    # 写入 0x000F 启用所有不可纠正错误上报
    $ setpci -s 0000:00:1f.6 ${EXP}+04.w=0x000F

    # 读 Corr Mask（2 字节）
    $ setpci -s 0000:00:1f.6 ${EXP}+08.w
    # 写入 0x0001 启用 Correctable Error 上报
    $ setpci -s 0000:00:1f.6 ${EXP}+08.w=0x0001

 3. 验证生效  
    再次：
      $ lspci -vv -s 0000:00:1f.6
    应看到 UEMsk、CEMsk 字段值已更新。

四、全局禁用/启用 AER  
 1. 禁用（kernel 参数）  
    在 /etc/default/grub 添加：
      GRUB_CMDLINE_LINUX="… pci=noaer …"
    更新并重启。

 2. 启用（移除 pci=noaer）并重启，或：  
    $ rmmod aer; modprobe aer

五、脚本化示例  
```bash
# 开启所有 AER
enable_aer() {
  dev=$1       # e.g. 0000:00:1f.6
  EXP=$(lspci -xxx -s $dev | grep -m1 "Exp Cap" | awk '{print $3}')
  setpci -s $dev ${EXP}+04.w=0x000F
  setpci -s $dev ${EXP}+08.w=0x0001
}
# 关闭所有 AER
disable_aer() {
  dev=$1
  # mask 全清
  EXP=$(lspci -xxx -s $dev | grep -m1 "Exp Cap" | awk '{print $3}')
  setpci -s $dev ${EXP}+04.w=0x0000
  setpci -s $dev ${EXP}+08.w=0x0000
}
*/

package pcie