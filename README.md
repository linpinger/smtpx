# smtpx 发文件到自己邮箱

- **缘起:** 现在的网络环境对小开发者极其不友好，没有一个可以直链存放文件的地方，有也会因为某种原因访问困难，或者要备案，既然如此，那就不直链了，使用一个邮箱来进行存储，标题就是文件名，文件放到附件中

- **说明:** 目前使用go语言开发出了原型，使用smtp协议发送邮件 (https://github.com/linpinger/smtpx) ，使用pop3协议 (https://github.com/linpinger/popx) 查看标题下载并自动提取附件,删除邮件等操作

- 由于go的特性,可以跨平台使用,目前pc,安卓手机都已实现基本目标,后续只需要简化操作流程,或使用imap协议进行更高级的管理文件,目标是自动分布式存储个人所有文件,只要邮箱够多^_^

- **编译:** 参见 (http://linpinger.olsoul.com/usr/2017-06-12_golang.html)  下的一般编译方法

- 本项目改自: (https://github.com/WillCastor/smtpx)

- 主要改进了: 

  - 添加: (https://github.com/stvoidit/gosmtp)  里面复制过来的 `generateBoundary()`
  - 添加: (https://github.com/jordan-wright/email) 里面的 `base64Wrap()` ，把附件base64格式化等长行，比较符合标准，方便 popx 解析
  - 修改: `buildAttachments()` 使附件文件名符合一般邮箱里的规则，方便 popx 解析

## 日志

- 2023-07-07: 第一版，基本能用

