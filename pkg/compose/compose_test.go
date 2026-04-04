package compose

import (
        "testing"
)

func TestValidate(t *testing.T) {
        tests := []struct {
                name    string
                cfg     VMConfig
                wantErr bool
        }{
                {
                        name: "valid config",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "MyUser",
                                Password: "MyPass",
                        },
                        wantErr: false,
                },
                {
                        name: "valid RAM in megabytes",
                        cfg: VMConfig{
                                RAMSize:  "512M",
                                CPUCores: "2",
                                DiskSize: "32G",
                                Version:  "10",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: false,
                },
                {
                        name:    "all empty fields",
                        cfg:     VMConfig{},
                        wantErr: true,
                },
                {
                        name: "invalid RAM format",
                        cfg: VMConfig{
                                RAMSize:  "4GB",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "CPU cores zero",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "0",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "CPU cores too high",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "65",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "CPU cores not a number",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "abc",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "invalid disk format",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "big",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "empty version",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "empty username",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "empty password",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "",
                        },
                        wantErr: true,
                },
        }

        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        err := Validate(&tt.cfg)
                        if (err != nil) != tt.wantErr {
                                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
                        }
                })
        }
}
