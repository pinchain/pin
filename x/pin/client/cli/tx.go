package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pinchain/pinchain/x/pin/types"
)

// GetTxCmd returns the transaction commands of the x/pin module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "PIN identity transactions",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdRegisterPIN(),
		CmdSendByPIN(),
	)

	return cmd
}

// CmdRegisterPIN registers a PIN for the signing account.
func CmdRegisterPIN() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [pin]",
		Short: "Register a PIN owned by the signing account",
		Long: fmt.Sprintf(`Register a PIN (format %s over alphabet %s) for the signing account.

If [pin] is omitted, a PIN is generated locally using cryptographically secure
randomness. A PIN is a public identifier only: it never authorizes spending.`,
			"XXX-XXXX", types.PINAlphabet),
		Example: "pinchaind tx pin register A7K-92XM --from alice",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var pin string
			if len(args) == 1 {
				if pin, err = types.NormalizeAndValidatePIN(args[0]); err != nil {
					return err
				}
			} else {
				if pin, err = types.GeneratePIN(); err != nil {
					return err
				}
				cmd.Printf("generated pin: %s\n", pin)
			}

			msg := types.NewMsgRegisterPIN(clientCtx.GetFromAddress().String(), pin)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdSendByPIN sends coins to the owner of a PIN.
func CmdSendByPIN() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-by-pin [recipient-pin] [amount]",
		Short: "Send tokens to the owner of a PIN",
		Long: fmt.Sprintf(`Resolve [recipient-pin] on chain and transfer [amount] to its owner.

Amounts may be given in base units (25000000%s) or display units (25%s).`,
			types.BaseDenom, types.DisplayDenom),
		Example: "pinchaind tx pin send-by-pin B4M-81QZ 25PIN --from alice",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			pin, err := types.NormalizeAndValidatePIN(args[0])
			if err != nil {
				return err
			}

			amount, err := types.ParseAmount(args[1])
			if err != nil {
				return err
			}

			msg := types.NewMsgSendByPIN(clientCtx.GetFromAddress().String(), pin, amount)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
