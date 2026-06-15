package identity

import "github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"

func SaveIdentity(paths config.Paths, identity Identity) error {
	if err := paths.Ensure(); err != nil {
		return err
	}
	if err := identity.Validate(); err != nil {
		return err
	}
	return config.SaveJSON(paths.IdentityPath(), identity, config.FileMode)
}

func LoadIdentity(paths config.Paths) (Identity, error) {
	var identity Identity
	if err := config.LoadJSON(paths.IdentityPath(), &identity); err != nil {
		return Identity{}, err
	}
	if err := identity.Validate(); err != nil {
		return Identity{}, err
	}
	return identity, nil
}

func SaveNetwork(paths config.Paths, network Network) error {
	if err := paths.Ensure(); err != nil {
		return err
	}
	if err := network.Validate(); err != nil {
		return err
	}
	return config.SaveJSON(paths.NetworkPath(), network, config.FileMode)
}

func LoadNetwork(paths config.Paths) (Network, error) {
	var network Network
	if err := config.LoadJSON(paths.NetworkPath(), &network); err != nil {
		return Network{}, err
	}
	if err := network.Validate(); err != nil {
		return Network{}, err
	}
	return network, nil
}
