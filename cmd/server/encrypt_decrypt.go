package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/relaxyabc/mcp-dbquery/src/crypto"
)

// encrypt.go - 加密命令处理器

// encryptCommand 加密子命令定义
func encryptCommand() *cli.Command {
	return &cli.Command{
		Name:  "encrypt",
		Usage: "加密明文密码，输出密文格式（用于配置文件）",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Usage:   "自定义加密密钥（可选，默认使用环境变量 DBQUERY_ENCRYPTION_KEY）",
			},
			&cli.StringFlag{
				Name:    "key-file",
				Aliases: []string{"f"},
				Usage:   "密钥文件路径（可选）",
			},
		},
		Action: encryptAction,
	}
}

// encryptAction 加密命令执行逻辑
func encryptAction(ctx context.Context, cmd *cli.Command) error {
	// 获取密码参数（第一个位置参数）
	args := cmd.Args()
	if args.Len() == 0 {
		return fmt.Errorf("密码不能为空")
	}
	password := args.First()

	// 获取加密密钥
	key, err := getEncryptionKey(cmd)
	if err != nil {
		return err
	}

	// 加密密码
	ciphertext, err := crypto.EncryptPassword(key, password)
	if err != nil {
		return fmt.Errorf("加密失败: %s", err)
	}

	// 输出密文（用于配置文件）
	fmt.Println(ciphertext)
	return nil
}

// decrypt.go - 解密命令处理器

// decryptCommand 解密子命令定义
func decryptCommand() *cli.Command {
	return &cli.Command{
		Name:  "decrypt",
		Usage: "解密密文密码，输出明文（用于验证配置）",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Usage:   "自定义解密密钥（可选，默认使用环境变量 DBQUERY_ENCRYPTION_KEY）",
			},
			&cli.StringFlag{
				Name:    "key-file",
				Aliases: []string{"f"},
				Usage:   "密钥文件路径（可选）",
			},
		},
		Action: decryptAction,
	}
}

// decryptAction 解密命令执行逻辑
func decryptAction(ctx context.Context, cmd *cli.Command) error {
	// 获取密文参数（第一个位置参数）
	args := cmd.Args()
	if args.Len() == 0 {
		return fmt.Errorf("密文不能为空")
	}
	ciphertext := args.First()

	// 获取解密密钥
	key, err := getEncryptionKey(cmd)
	if err != nil {
		return err
	}

	// 解密密码
	plaintext, err := crypto.DecryptPassword(key, ciphertext)
	if err != nil {
		return fmt.Errorf("解密失败: %s", err)
	}

	// 输出明文（用于验证）
	fmt.Println(plaintext)
	return nil
}

// keyutils.go - 密钥获取辅助函数

// getEncryptionKey 从命令参数或环境变量获取加密密钥
func getEncryptionKey(cmd *cli.Command) (string, error) {
	// 优先使用命令行指定的密钥
	key := cmd.String("key")
	if key != "" {
		return key, nil
	}

	// 其次使用密钥文件
	keyFile := cmd.String("key-file")
	if keyFile != "" {
		key, err := crypto.GetEncryptionKeyFromFile(keyFile)
		if err != nil {
			return "", err
		}
		return key, nil
	}

	// 最后使用环境变量
	key, err := crypto.GetEncryptionKey()
	if err != nil {
		return "", err
	}
	return key, nil
}